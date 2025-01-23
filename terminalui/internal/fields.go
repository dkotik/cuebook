package internal

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/patch"
	"github.com/dkotik/cuebook/terminalui/field"
	"github.com/dkotik/cuebook/terminalui/list"
	"github.com/dkotik/cuebook/terminalui/textarea"
	"github.com/dkotik/cuebook/terminalui/window"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

type (
	fieldHighlighted int
	fieldListCards   []tea.Model

	updateFieldPatch struct {
		patch.Patch
		Entry cuebook.Entry
	}

	deleteEntryPatch struct {
		patch.Patch
		Entry cuebook.Entry
	}
)

type FieldList struct {
	tea.Model

	state    patch.Result
	entry    cuebook.Entry
	selected int
}

func (l FieldList) Init() (_ tea.Model, cmd tea.Cmd) {
	l.Model = list.New()
	// l.selected = -1
	return l, nil
}

func (l FieldList) Update(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	switch msg := msg.(type) {
	case fieldHighlighted:
		l.selected = int(msg)
		return l, nil
	case patch.Result:
		// if !l.state.IsEqual(msg) { }
		l.state = msg
		if _, ok := msg.LastChange.(updateFieldPatch); ok {
			return l, func() tea.Msg { return window.BackEvent{} }
		}
		if _, ok := msg.LastChange.(deleteEntryPatch); ok {
			return l, func() tea.Msg { return window.BackEvent{} }
		}
		return l, nil
	case cuebook.Entry:
		l.entry = msg
		return l, LoadFields(l.state.Source, msg)
	case fieldListCards:
		l.Model, cmd = l.Model.Init()
		var setCmd tea.Cmd
		l.Model, setCmd = l.Model.Update(list.SetItems(msg...)())
		return l, tea.Sequence(cmd, setCmd, tea.RequestWindowSize())
	default:
		l.Model, cmd = l.Model.Update(msg)
		return l, cmd
	}
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

func LoadFields(source []byte, entry cuebook.Entry) tea.Cmd {
	return func() tea.Msg {
		return window.TranslatableFunc(func(lc *i18n.Localizer) tea.Cmd {
			fields := make([]tea.Model, 0, len(entry.Fields)+len(entry.Details)+1)
			// fields = append(fields, list.Title{
			// 	Text:  entry.GetTitle() + fmt.Sprintf(" › %d/%d", index+1, total),
			// 	Style: lipgloss.NewStyle().Bold(true).Align(lipgloss.Left).Foreground(lipgloss.BrightRed),
			// })
			for _, f := range entry.Fields {
				fields = append(fields, field.New(f.Name, f.String(), displayFieldForm(source, entry, f)))
			}
			for _, f := range entry.Details {
				fields = append(fields, field.New(f.Name, f.String(), displayFieldForm(source, entry, f)))
			}
			rmLabel := lc.MustLocalize(&i18n.LocalizeConfig{
				DefaultMessage: &i18n.Message{
					ID:    "bookEntryDelete",
					Other: "Delete entry",
				},
			})
			return func() tea.Msg {
				// bWrapper, bCapture := NewPatchCloser("entryDelete")
				return fieldListCards(append(fields, list.NewButton(rmLabel, func() tea.Msg {
					p, err := patch.DeleteFromStructList(source, entry.Value)
					if err != nil {
						return err
					}
					return deleteEntryPatch{
						Patch: p,
						Entry: entry,
					}
				})))
			}
		})
	}
}
