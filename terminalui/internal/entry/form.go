package entry

import (
	"bytes"
	"slices"

	"cuelang.org/go/cue"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/patch"
	"github.com/dkotik/cuebook/terminalui/event"
	listForm "github.com/dkotik/cuebook/terminalui/form"
	"github.com/dkotik/cuebook/terminalui/list"
	"github.com/dkotik/cuebook/terminalui/textarea"
	"github.com/dkotik/cuebook/terminalui/window"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

type (
	fieldHighlighted int
	fieldListCards   struct {
		Cards           []tea.Model
		KnownFieldNames []string
	}

	fieldChangedEvent struct {
		Name     string
		Value    string
		Original string
	}

	applyFieldChanges struct{}

	updateFieldPatch struct {
		patch.Patch
		Entry cuebook.Entry
	}

	swapEntriesPatch struct {
		patch.Patch
		// Target patch.ByteAnchor
	}

	swapFieldsPatch struct {
		patch.Patch
	}

	deleteEntryPatch struct {
		patch.Patch
		Entry cuebook.Entry
	}
)

func LoadFields(cueEntry cuebook.Entry) tea.Cmd {
	return func() tea.Msg {
		expectedTotal := len(cueEntry.Fields) + len(cueEntry.Details) + 3
		knownFieldNames := make([]string, 0, expectedTotal)
		fields := make([]tea.Model, 0, expectedTotal)
		// fields = append(fields, list.Title{
		// 	Text:  entry.GetTitle() + fmt.Sprintf(" › %d/%d", index+1, total),
		// 	Style: lipgloss.NewStyle().Bold(true).Align(lipgloss.Left).Foreground(lipgloss.BrightRed),
		// })
		for _, f := range cueEntry.Fields {
			fields = append(fields, createField(f))
			knownFieldNames = append(knownFieldNames, f.Name)
		}
		for _, f := range cueEntry.Details {
			fields = append(fields, createField(f))
			knownFieldNames = append(knownFieldNames, f.Name)
		}
		if cueEntry.Value.Allows(cue.AnyString) {
			fields = append(fields, NewExtendButton())
		}
		return fieldListCards{
			Cards: append(
				fields,
				NewSaveButton(),
				NewDeleteButton(),
			),
			KnownFieldNames: knownFieldNames,
		}
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
}

func (f form) Init() (tea.Model, tea.Cmd) {
	var (
		listInit  tea.Cmd
		buildInit tea.Cmd
	)

	f.Model, listInit = listForm.New().Init()
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
			return Update(func(state patch.Result) (patch.Patch, error) {
				p, err := patch.MergeFieldValues(state.Source, f.Entry.Value, f.Changes)
				if err != nil {
					return nil, err
				}
				return updateFieldPatch{
					Patch: p,
					Entry: f.Entry,
				}, nil
			})
		}
	case list.SwapOrderEvent:
		return f, func() tea.Msg {
			return Update(func(state patch.Result) (patch.Patch, error) {
				a, err := f.Entry.GetField(msg.CurrentIndex)
				if err != nil {
					return nil, err
				}
				b, err := f.Entry.GetField(msg.DesiredIndex)
				if err != nil {
					return nil, err
				}
				p, err := patch.SwapEntries(state.Source, a.Value, b.Value)
				if err != nil {
					return nil, err
				}
				return swapFieldsPatch{Patch: p}, nil
			})
		}
	case deleteEvent:
		return f, func() tea.Msg {
			return Update(func(state patch.Result) (patch.Patch, error) {
				p, err := patch.DeleteFromStructList(state.Source, f.Entry.Value)
				if err != nil {
					return nil, err
				}
				return deleteEntryPatch{ // TODO: restore matching to patch.Result.LastChange type
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
	case extendEvent:
		if slices.Index(f.KnownFields, msg.Name) >= 0 {
			return f, window.NewFlashMessage(window.FlashMessageKindWarning, &i18n.LocalizeConfig{
				DefaultMessage: &i18n.Message{
					ID:    "bookEntryDuplicateFieldName",
					Other: "Field `{{ .Name }}` already exists.",
				},
				TemplateData: map[string]any{
					"Name": msg.Name,
				},
			})
		}
		f.KnownFields = append(f.KnownFields, msg.Name)
		f.Model, cmd = f.Model.Update(list.AddItems(createField(cuebook.Field{
			Name: msg.Name,
		}))())
		return f, tea.Sequence(
			cmd,
			tea.RequestWindowSize(),
			func() tea.Msg { return window.BackEvent{} },
		)
	case fieldListCards:
		f.KnownFields = msg.KnownFieldNames
		f.Model, cmd = listForm.New().Init()
		initCmd := event.PropagateInit(msg.Cards)
		var setCmd tea.Cmd
		f.Model, setCmd = f.Model.Update(list.SetItems(msg.Cards...)())
		return f, tea.Sequence(cmd, setCmd, initCmd, tea.RequestWindowSize(), window.RequestLocalizer())
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
	case patch.Result:
		switch p := msg.LastChange.(type) {
		case swapFieldsPatch:
			return f, func() tea.Msg {
				delta := p.Difference().Content
				// duplicates := p.Difference().PreceedingDuplicates // they do not matter since bytes identical
				if len(delta) > 0 {
					for entry, err := range msg.Document.EachEntry() {
						if err != nil {
							return err
						}
						at, err := patch.NewByteRange(entry.Value)
						if err != nil {
							return err
						}
						// TODO: this currently does not work
						if bytes.Equal(delta, msg.Source[at.Head:at.Tail]) {
							return LoadFields(entry)
						}
					}
				}
				// return errors.New("no entries matched")
				return window.BackEvent{}
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

func displayFieldForm(source []byte, entry cuebook.Entry, field cuebook.Field) tea.Cmd {
	return func() tea.Msg {
		form, err := textarea.New(
			textarea.WithLabel(field.Name),
			textarea.WithValue(field.String()),
			textarea.WithOnSubmitCommand(func(value string) tea.Cmd {
				return func() tea.Msg {
					p, err := patch.UpdateFieldValue(source, entry.Value, field.Value, value)
					if err != nil {
						return err
					}
					return updateFieldPatch{
						Patch: p,
						Entry: entry,
					}
				}
			}),
		)
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
