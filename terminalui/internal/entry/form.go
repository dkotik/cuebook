package entry

import (
	"slices"

	tea "charm.land/bubbletea/v2"
	"cuelang.org/go/cue"
	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/metadata"
	"github.com/dkotik/cuebook/terminalui/event"
	listForm "github.com/dkotik/cuebook/terminalui/internal/form"
	"github.com/dkotik/cuebook/terminalui/internal/list"
	"github.com/dkotik/cuebook/terminalui/internal/textarea"
	"github.com/dkotik/cuebook/terminalui/window"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

type (
	fieldHighlighted int
	fieldListCards   struct {
		Cards           []tea.Model
		KnownFieldNames []string
		Entry           cuebook.Entry
		Order           []orderedField
		SelectedIndex   int
	}

	// orderedField is a field card in display order. Only fields written in
	// the entry source can be reordered, and only within their own group.
	orderedField struct {
		cuebook.Field
		Detail   bool
		InSource bool
	}

	// swapFieldsEvent has its own type so the entry list below the entry view
	// does not mistake it for a request to reorder entries.
	swapFieldsEvent list.SwapOrderEvent

	fieldChangedEvent struct {
		Name     string
		Value    string
		Original string
	}

	applyFieldChanges struct{}

	updateFieldPatch struct {
		cuebook.Patch
		Entry cuebook.Entry
	}

	swapEntriesPatch struct {
		cuebook.Patch
		// Target cuebook.ByteAnchor
	}

	swapFieldsPatch struct {
		cuebook.Patch
		SelectedIndex int
	}
)

func (p updateFieldPatch) UnwrapPatch() cuebook.Patch { return p.Patch }
func (p swapEntriesPatch) UnwrapPatch() cuebook.Patch { return p.Patch }
func (p swapFieldsPatch) UnwrapPatch() cuebook.Patch  { return p.Patch }

func LoadFields(cueEntry cuebook.Entry) tea.Cmd {
	return loadFields(cueEntry, 0)
}

func loadFields(cueEntry cuebook.Entry, selectedIndex int) tea.Cmd {
	return func() tea.Msg {
		entryFields := withoutIdentifierField(cueEntry, cueEntry.Fields)
		detailFields := withoutIdentifierField(cueEntry, cueEntry.Details)
		expectedTotal := len(entryFields) + len(detailFields) + 3
		knownFieldNames := make([]string, 0, expectedTotal)
		fields := make([]tea.Model, 0, expectedTotal)
		// fields = append(fields, list.Title{
		// 	Text:  entry.Title + fmt.Sprintf(" › %d/%d", index+1, total),
		// 	Style: lipgloss.NewStyle().Bold(true).Align(lipgloss.Left).Foreground(lipgloss.BrightRed),
		// })
		order := append(
			sourceOrder(cueEntry, entryFields, false),
			sourceOrder(cueEntry, detailFields, true)...,
		)
		for _, f := range order {
			fields = append(fields, createField(f.Field))
			knownFieldNames = append(knownFieldNames, f.Name)
		}
		if metadata.IsOpen(cueEntry.Value) {
			fields = append(fields, NewExtendButton())
		}
		return fieldListCards{
			Cards: append(
				fields,
				NewSaveButton(),
				NewArchiveButton(),
			),
			KnownFieldNames: knownFieldNames,
			Entry:           cueEntry,
			Order:           order,
			SelectedIndex:   selectedIndex,
		}
	}
}

func withoutIdentifierField(entry cuebook.Entry, fields []cuebook.Field) []cuebook.Field {
	identifierFieldName := entry.UniqueIdentifierFieldName()
	if identifierFieldName == "" {
		return fields
	}
	filtered := make([]cuebook.Field, 0, len(fields))
	for _, field := range fields {
		if field.Name != identifierFieldName {
			filtered = append(filtered, field)
		}
	}
	return filtered
}

// sourceOrder sorts fields by where they are written in the entry, so that
// reordering the source also reorders the cards. Fields that are not written
// in the entry, such as unset schema fields, keep their order after the rest.
func sourceOrder(cueEntry cuebook.Entry, fields []cuebook.Field, detail bool) []orderedField {
	entryRange, entryErr := cuebook.NewByteRange(cueEntry.Value)
	heads := make(map[string]int, len(fields))
	ordered := make([]orderedField, 0, len(fields))
	for _, field := range fields {
		f := orderedField{Field: field, Detail: detail}
		if r, err := cuebook.NewByteRange(field.Value); err == nil && entryErr == nil &&
			r.Head >= entryRange.Head && r.Tail <= entryRange.Tail {
			f.InSource = true
			heads[field.Name] = r.Head
		}
		ordered = append(ordered, f)
	}
	slices.SortStableFunc(ordered, func(a, b orderedField) int {
		switch {
		case a.InSource && b.InSource:
			return heads[a.Name] - heads[b.Name]
		case a.InSource:
			return -1
		case b.InSource:
			return 1
		}
		return 0
	})
	return ordered
}

// loadFieldsSelecting loads the fields of an entry and selects the named one,
// such as a field that was just added, so it can be edited right away.
func loadFieldsSelecting(cueEntry cuebook.Entry, name string) tea.Cmd {
	return func() tea.Msg {
		msg := loadFields(cueEntry, 0)()
		cards, ok := msg.(fieldListCards)
		if !ok {
			return msg
		}
		for index, field := range cards.Order {
			if field.Name == name {
				cards.SelectedIndex = index
				break
			}
		}
		return cards
	}
}

func NewForm(book cuebook.Book, ref cuebook.EntryRef) tea.Cmd {
	entry := ref.Entry()
	return tea.Sequence(
		func() tea.Msg {
			return window.SwitchTo(newForm(book, ref))
		},
		LoadFields(entry),
	)
}

func newForm(book cuebook.Book, ref cuebook.EntryRef) form {
	return form{
		Model: newFieldList(), Book: book, EntryRef: ref,
		Entry: ref.Entry(), Changes: make(map[string]string),
	}
}

type form struct {
	tea.Model

	Book        cuebook.Book
	EntryRef    cuebook.EntryRef
	Entry       cuebook.Entry
	Fields      []tea.Model
	Buttons     []tea.Model
	Changes     map[string]string
	KnownFields []string
	Order       []orderedField
}

func newFieldList() tea.Model {
	return listForm.New(listForm.Configuration{
		OnSwapOrder: func(e list.SwapOrderEvent) tea.Msg { return swapFieldsEvent(e) },
	})
}

// swappable returns the two fields at the list indexes when they can trade
// places in the entry source.
func (f form) swappable(current, desired int) (a, b orderedField, ok bool) {
	if current < 0 || desired < 0 || current >= len(f.Order) || desired >= len(f.Order) {
		return a, b, false
	}
	a, b = f.Order[current], f.Order[desired]
	return a, b, a.InSource && b.InSource && a.Detail == b.Detail
}

func (f form) Init() tea.Cmd {
	return tea.Batch(
		f.Model.Init(),
		event.PropagateInit(f.Fields),
		event.PropagateInit(f.Buttons),
		window.RequestLocalizer(),
	)
}

func (f form) buildForm() (_ form, cmd tea.Cmd) {
	f.Model, cmd = f.Model.Update(list.SetItems(
		append(f.Fields, f.Buttons...)...)())
	return f, tea.Batch(cmd, window.RequestWindowSize(), window.RequestLocalizer())
}

func (f form) Update(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	switch msg := msg.(type) {
	case listForm.SaveChangesEvent:
		values := make(map[string]string, len(f.Changes))
		var removals []string
		for name, value := range f.Changes {
			if value == "" && emptyFieldShouldBeRemoved(f.Entry, name) {
				removals = append(removals, name)
				continue
			}
			values[name] = value
		}
		return f, func() tea.Msg {
			return Update(func(_ cuebook.Result) (cuebook.Patch, error) {
				p, err := f.Book.MergeFieldValuesAndRemove(f.EntryRef, values, removals)
				if err != nil {
					return nil, err
				}
				return updateFieldPatch{
					Patch: p,
					Entry: f.Entry,
				}, nil
			})
		}
	case swapFieldsEvent:
		if len(f.Changes) > 0 {
			return f, window.NewFlashMessage(window.FlashMessageKindWarning, &i18n.LocalizeConfig{
				DefaultMessage: &i18n.Message{
					ID:    "bookEntryReorderUnsavedChangesWarning",
					Other: "Save or discard your changes before reordering fields.",
				},
			})
		}
		a, b, ok := f.swappable(msg.CurrentIndex, msg.DesiredIndex)
		if !ok {
			return f, nil
		}
		selected := msg.DesiredIndex
		return f, func() tea.Msg {
			return Update(func(state cuebook.Result) (cuebook.Patch, error) {
				p, err := f.Book.SwapFields(f.EntryRef, a.Name, b.Name)
				if err != nil {
					return nil, err
				}
				return swapFieldsPatch{Patch: p, SelectedIndex: selected}, nil
			})
		}
	case archiveEvent:
		return f, func() tea.Msg {
			return ArchiveEntryEvent{Book: f.Book, Entry: f.EntryRef, ReturnToList: true}
		}
	case fieldChangedEvent:
		if msg.Value == msg.Original {
			delete(f.Changes, msg.Name)
			if len(f.Changes) == 0 {
				f.Model, cmd = f.Model.Update(cancelEvent{})
			}
		} else {
			f.Changes[msg.Name] = msg.Value
			f.Model, cmd = f.Model.Update(msg)
		}
		return f, cmd
	case extendRequestEvent:
		// Adding a field rewrites the entry, which would discard unsaved changes.
		if len(f.Changes) > 0 {
			return f, window.NewFlashMessage(window.FlashMessageKindWarning, &i18n.LocalizeConfig{
				DefaultMessage: &i18n.Message{
					ID:    "bookEntryExtendUnsavedChangesWarning",
					Other: "Save or discard your changes before adding a field.",
				},
			})
		}
		knownFields := slices.Clone(f.KnownFields)
		return f, func() tea.Msg { return window.SwitchTo(newNewFieldForm(knownFields)) }
	case addFieldEvent:
		return f, tea.Sequence(
			func() tea.Msg { return window.BackEvent{} },
			func() tea.Msg {
				return Update(func(state cuebook.Result) (cuebook.Patch, error) {
					p, err := f.Book.AddField(f.EntryRef, msg.Name, msg.Kind, msg.Value)
					if err != nil {
						return nil, err
					}
					return addFieldPatch{Patch: p, Name: msg.Name}, nil
				})
			},
		)
	case fieldListCards:
		f.KnownFields = msg.KnownFieldNames
		f.Entry = msg.Entry
		f.Order = msg.Order
		f.Model = newFieldList()
		cmd = f.Model.Init()
		initCmd := event.PropagateInit(msg.Cards)
		var setCmd, selectCmd tea.Cmd
		f.Model, setCmd = f.Model.Update(list.SetItems(msg.Cards...)())
		f.Model, selectCmd = f.Model.Update(list.ApplySelection(msg.SelectedIndex)())
		return f, tea.Sequence(cmd, setCmd, initCmd, selectCmd, window.RequestWindowSize(), window.RequestLocalizer())
	case tea.KeyPressMsg:
		if msg.Key().Code == tea.KeyEscape && len(f.Changes) > 0 {
			return f, window.NewFlashMessage(window.FlashMessageKindWarning, &i18n.LocalizeConfig{
				DefaultMessage: &i18n.Message{
					ID:    "bookEntryUnsavedChangesWarning",
					One:   "There is one unsaved change.",
					Other: "There are {{ .Count }} changes that were not saved.",
				},
				TemplateData: map[string]any{
					"Count": len(f.Changes),
				},
				PluralCount: len(f.Changes),
			})
		}
	case ArchiveEntryCompletedEvent:
		return f, window.NewFlashMessage(window.FlashMessageKindSuccess, &i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "flashMessageEntryArchived",
				Other: "Entry `{{.Title}}` archived.",
			},
			TemplateData: map[string]interface{}{
				"Title": msg.Title,
			},
		})
	case cuebook.Result:
		switch p := msg.LastChange.(type) {
		case addFieldPatch:
			return f, tea.Batch(
				window.NewFlashMessage(window.FlashMessageKindSuccess, &i18n.LocalizeConfig{
					DefaultMessage: &i18n.Message{
						ID:    "flashMessageFieldAdded",
						Other: "Field `{{ .Name }}` added.",
					},
					TemplateData: map[string]any{"Name": p.Name},
				}),
				func() tea.Msg {
					entry, ok := entryContaining(msg, p.Difference())
					if !ok {
						return window.BackEvent{}
					}
					return loadFieldsSelecting(entry, p.Name)()
				},
			)
		case swapFieldsPatch:
			return f, func() tea.Msg {
				entry, ok := entryContaining(msg, p.Difference())
				if !ok {
					return window.BackEvent{}
				}
				return loadFields(entry, p.SelectedIndex)()
			}
		case updateFieldPatch:
			return f, tea.Batch(
				window.NewFlashMessage(window.FlashMessageKindSuccess, &i18n.LocalizeConfig{
					DefaultMessage: &i18n.Message{
						ID:    "flashMessageEntryUpdated",
						Other: "Entry `{{.Title}}` updated.",
					},
					TemplateData: map[string]interface{}{
						"Title": p.Entry.Title, // TODO: old title, obtain new using swapFieldPatch algorithm
					},
				}),
				func() tea.Msg { return window.BackEvent{} },
			)

		}
	}
	f.Model, cmd = f.Model.Update(msg)
	return f, cmd
}

// entryContaining finds the entry affected by the patch difference. Result.Match
// resolves identified patches by UUID and falls back to byte anchors for
// anonymous entries.
func entryContaining(r cuebook.Result, difference cuebook.ByteAnchor) (cuebook.Entry, bool) {
	return r.Match(difference.Content)
}

func displayFieldForm(book cuebook.Book, ref cuebook.EntryRef, field cuebook.Field) tea.Cmd {
	return func() tea.Msg {
		form, err := textarea.New(textarea.Configuration{
			Label: field.Name,
			Value: field.String(),
			OnSubmit: func(value string) tea.Cmd {
				return func() tea.Msg {
					p, err := book.UpdateFieldValue(ref, field.Name, value)
					if err != nil {
						return err
					}
					return updateFieldPatch{
						Patch: p,
						Entry: ref.Entry(),
					}
				}
			},
		})
		if err != nil {
			return err
		}
		return window.SwitchTo(form)
	}
}

func emptyFieldShouldBeRemoved(entry cuebook.Entry, name string) bool {
	for _, definition := range entry.FieldDefinitions {
		if definition.Selector.Unquoted() == name {
			return definition.Selector.ConstraintType() == cue.OptionalConstraint
		}
	}
	return true
}

func createField(f cuebook.Field) tea.Model {
	return listForm.NewField(f.Name, f.String(), func(updated string) tea.Cmd {
		original := f.String()
		return tea.Batch(
			func() tea.Msg {
				return fieldChangedEvent{
					Name:     f.Name,
					Value:    updated,
					Original: original,
				}
			},
		)
	})
}
