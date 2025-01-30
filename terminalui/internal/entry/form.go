package entry

import (
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
	fieldListCards   []tea.Model

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

func NewForm(state patch.Result) tea.Cmd {
	return func() tea.Msg {
		return window.SwitchTo(form{state: state})
	}
}

type form struct {
	tea.Model

	changes  map[string]string
	state    patch.Result
	entry    cuebook.Entry
	selected int
}

func (f form) Init() (_ tea.Model, cmd tea.Cmd) {
	f.Model, cmd = listForm.New().Init()
	f.changes = make(map[string]string)
	return f, cmd
}

func (f form) Update(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	switch msg := msg.(type) {
	case listForm.SaveChangesEvent:
		return f, func() tea.Msg {
			p, err := patch.MergeFieldValues(f.state.Source, f.entry.Value, f.changes)
			if err != nil {
				return err
			}
			return updateFieldPatch{
				Patch: p,
				Entry: f.entry,
			}
		}
	case deleteEvent:
		return f, func() tea.Msg {
			p, err := patch.DeleteFromStructList(f.state.Source, f.entry.Value)
			if err != nil {
				return err
			}
			return deleteEntryPatch{
				Patch: p,
				Entry: f.entry,
			}
		}
	case fieldChangedEvent:
		if msg.Value == msg.Original {
			delete(f.changes, msg.Name)
			if len(f.changes) == 0 {
				f.Model, cmd = f.Model.Update(cancelEvent{})
			}
		} else {
			f.changes[msg.Name] = msg.Value
			f.Model, cmd = f.Model.Update(msg)
		}
		return f, cmd
	case extendEvent:
		return f, tea.Sequence(
			func() tea.Msg {
				return list.AddItems(createField(cuebook.Field{
					Name: msg.Name,
				}))()
			},
			tea.RequestWindowSize(),
			func() tea.Msg { return window.BackEvent{} },
		)
	case fieldHighlighted:
		f.selected = int(msg)
		return f, nil
	case patch.Result:
		// if !f.state.IsEqual(msg) { }
		f.state = msg
		if _, ok := msg.LastChange.(updateFieldPatch); ok {
			return f, func() tea.Msg { return window.BackEvent{} }
		}
		if _, ok := msg.LastChange.(deleteEntryPatch); ok {
			return f, func() tea.Msg { return window.BackEvent{} }
		}
		return f, nil
	case cuebook.Entry:
		f.entry = msg
		return f, LoadFields(f.state.Source, msg)
	case fieldListCards:
		f.Model, cmd = listForm.New().Init()
		initCmd := event.PropagateInit(msg)
		var setCmd tea.Cmd
		f.Model, setCmd = f.Model.Update(list.SetItems(msg...)())
		return f, tea.Sequence(cmd, setCmd, initCmd, tea.RequestWindowSize(), window.RequestLocalizer())
	case list.SwapOrderEvent:
		return f, func() tea.Msg {
			a, err := f.entry.GetField(msg.CurrentIndex)
			if err != nil {
				return err
			}
			b, err := f.entry.GetField(msg.DesiredIndex)
			if err != nil {
				return err
			}
			p, err := patch.SwapEntries(f.state.Source, a.Value, b.Value)
			if err != nil {
				return err
			}
			return swapFieldsPatch{Patch: p}
		}
	case tea.KeyMsg:
		if msg.Key().Code == tea.KeyEscape && len(f.changes) > 0 {
			return f, window.NewFlashMessage(window.FlashMessageKindWarning, &i18n.LocalizeConfig{
				DefaultMessage: &i18n.Message{
					ID:    "bookEntryUnsavedChangesWarning",
					One:   "There is one unsaved change.",
					Other: "There are {{ .Count }} changes that were not saved.",
				},
				TemplateData: map[string]any{
					"Count": len(f.changes),
				},
				PluralCount: len(f.changes),
			})
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

func LoadFields(source []byte, cueEntry cuebook.Entry) tea.Cmd {
	return func() tea.Msg {
		fields := make([]tea.Model, 0, len(cueEntry.Fields)+len(cueEntry.Details)+3)
		// fields = append(fields, list.Title{
		// 	Text:  entry.GetTitle() + fmt.Sprintf(" › %d/%d", index+1, total),
		// 	Style: lipgloss.NewStyle().Bold(true).Align(lipgloss.Left).Foreground(lipgloss.BrightRed),
		// })
		for _, f := range cueEntry.Fields {
			fields = append(fields, createField(f))
		}
		for _, f := range cueEntry.Details {
			fields = append(fields, createField(f))
		}
		if cueEntry.Value.Allows(cue.AnyString) {
			fields = append(fields, NewExtendButton())
		}
		return fieldListCards(append(
			fields,
			NewSaveButton(),
			NewDeleteButton(),
		))
	}
}
