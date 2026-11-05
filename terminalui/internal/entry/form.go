package entry

import (
	"slices"

	tea "github.com/charmbracelet/bubbletea/v2"
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

	deleteEntryPatch struct {
		cuebook.Patch
		Entry cuebook.Entry
	}
)

func LoadFields(cueEntry cuebook.Entry) tea.Cmd {
	return loadFields(cueEntry, 0)
}

func loadFields(cueEntry cuebook.Entry, selectedIndex int) tea.Cmd {
	return func() tea.Msg {
		expectedTotal := len(cueEntry.Fields) + len(cueEntry.Details) + 3
		knownFieldNames := make([]string, 0, expectedTotal)
		fields := make([]tea.Model, 0, expectedTotal)
		// fields = append(fields, list.Title{
		// 	Text:  entry.GetTitle() + fmt.Sprintf(" › %d/%d", index+1, total),
		// 	Style: lipgloss.NewStyle().Bold(true).Align(lipgloss.Left).Foreground(lipgloss.BrightRed),
		// })
		order := append(
			sourceOrder(cueEntry, cueEntry.Fields, false),
			sourceOrder(cueEntry, cueEntry.Details, true)...,
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
				NewDeleteButton(),
			),
			KnownFieldNames: knownFieldNames,
			Entry:           cueEntry,
			Order:           order,
			SelectedIndex:   selectedIndex,
		}
	}
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

func NewForm(entry cuebook.Entry) tea.Cmd {
	return tea.Sequence(
		func() tea.Msg {
			return window.SwitchTo(form{Entry: entry})
		},
		LoadFields(entry),
	)
}

type form struct {
	tea.Model

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

func (f form) Init() (tea.Model, tea.Cmd) {
	var (
		listInit  tea.Cmd
		buildInit tea.Cmd
	)

	f.Model, listInit = newFieldList().Init()
	f.Changes = make(map[string]string)
	fieldsInit := event.PropagateInit(f.Fields)
	buttonsInit := event.PropagateInit(f.Buttons)
	f, buildInit = f.buildForm()
	return f, tea.Batch(
		listInit,
		fieldsInit,
		buttonsInit,
		buildInit,
	)
}

func (f form) buildForm() (_ form, cmd tea.Cmd) {
	f.Model, cmd = f.Model.Update(list.SetItems(
		append(f.Fields, f.Buttons...)...)())
	return f, tea.Batch(cmd, tea.RequestWindowSize(), window.RequestLocalizer())
}

func (f form) Update(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	switch msg := msg.(type) {
	case listForm.SaveChangesEvent:
		return f, func() tea.Msg {
			return Update(func(state cuebook.Result) (cuebook.Patch, error) {
				p, err := cuebook.MergeFieldValues(state.Source, f.Entry.Value, f.Changes)
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
		entryValue := f.Entry.Value
		selected := msg.DesiredIndex
		return f, func() tea.Msg {
			return Update(func(state cuebook.Result) (cuebook.Patch, error) {
				p, err := cuebook.SwapFields(state.Source, entryValue, a.Name, b.Name)
				if err != nil {
					return nil, err
				}
				return swapFieldsPatch{Patch: p, SelectedIndex: selected}, nil
			})
		}
	case deleteEvent:
		return f, func() tea.Msg {
			return Update(func(state cuebook.Result) (cuebook.Patch, error) {
				p, err := cuebook.DeleteFromStructList(state.Source, f.Entry.Value)
				if err != nil {
					return nil, err
				}
				return deleteEntryPatch{ // TODO: restore matching to cuebook.Result.LastChange type
					Patch: p,
					Entry: f.Entry,
				}, nil
			})
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
		entryValue := f.Entry.Value
		return f, tea.Sequence(
			func() tea.Msg { return window.BackEvent{} },
			func() tea.Msg {
				return Update(func(state cuebook.Result) (cuebook.Patch, error) {
					p, err := cuebook.AddField(state.Source, entryValue, msg.Name, msg.Kind, msg.Value)
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
		f.Model, cmd = newFieldList().Init()
		initCmd := event.PropagateInit(msg.Cards)
		var setCmd, selectCmd tea.Cmd
		f.Model, setCmd = f.Model.Update(list.SetItems(msg.Cards...)())
		f.Model, selectCmd = f.Model.Update(list.ApplySelection(msg.SelectedIndex)())
		return f, tea.Sequence(cmd, setCmd, initCmd, selectCmd, tea.RequestWindowSize(), window.RequestLocalizer())
	case tea.KeyMsg:
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
						"Title": p.Entry.GetTitle(), // TODO: old title, obtain new using swapFieldPatch algorithm
					},
				}),
				func() tea.Msg { return window.BackEvent{} },
			)
		case deleteEntryPatch:
			return f, tea.Batch(
				window.NewFlashMessage(window.FlashMessageKindWarning, &i18n.LocalizeConfig{
					DefaultMessage: &i18n.Message{
						ID:    "flashMessageEntryDeleted",
						Other: "Entry `{{.Title}}` deleted.",
					},
					TemplateData: map[string]interface{}{
						"Title": p.Entry.GetTitle(),
					},
				}),
				func() tea.Msg { return window.BackEvent{} },
			)
		}
	}
	f.Model, cmd = f.Model.Update(msg)
	return f, cmd
}

// entryContaining finds the entry that holds the anchored bytes, such as a
// field that was just moved. [cuebook.Result.Match] does the reverse: it
// finds an entry inside the anchored bytes.
func entryContaining(r cuebook.Result, anchor cuebook.ByteAnchor) (cuebook.Entry, bool) {
	target, err := anchor.Match(r.Source)
	if err != nil {
		return cuebook.Entry{}, false
	}
	for entry, err := range r.Document.EachEntry() {
		if err != nil {
			continue
		}
		at, err := cuebook.NewByteRange(entry.Value)
		if err == nil && at.Head <= target.Head && target.Tail <= at.Tail {
			return entry, true
		}
	}
	return cuebook.Entry{}, false
}

func displayFieldForm(source []byte, entry cuebook.Entry, field cuebook.Field) tea.Cmd {
	return func() tea.Msg {
		form, err := textarea.New(textarea.Configuration{
			Label: field.Name,
			Value: field.String(),
			OnSubmit: func(value string) tea.Cmd {
				return func() tea.Msg {
					p, err := cuebook.UpdateFieldValue(source, entry.Value, field.Value, value)
					if err != nil {
						return err
					}
					return updateFieldPatch{
						Patch: p,
						Entry: entry,
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
