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

	fieldViewClosingPatch struct {
		patch.Patch
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
		if _, ok := msg.LastChange.(fieldViewClosingPatch); ok {
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
	case tea.KeyMsg:
		l.Model, cmd = l.Model.Update(msg)
		switch msg.Key().Code {
		case tea.KeyEnter:
			return l, displayFieldForm(l.state.Source, l.entry, l.selected)
		}
		return l, NewListItemHighlightAdaptor[fieldHighlighted](cmd)
	default:
		l.Model, cmd = l.Model.Update(msg)
		return l, cmd
	}
}

func displayFieldForm(source []byte, entry cuebook.Entry, index int) tea.Cmd {
	return func() tea.Msg {
		field, err := entry.GetField(index)
		if err != nil {
			return err
		}
		formWrapper, patchWrapper := NewPatchCloser("fieldPatch")
		form, err := textarea.New(
			textarea.WithLabel(field.Name),
			textarea.WithValue(field.String()),
			textarea.WithOnSubmitCommand(func(value string) tea.Cmd {
				return func() tea.Msg {
					p, err := patch.UpdateFieldValue(source, entry.Value, field.Value, value)
					if err != nil {
						return err
					}
					return patchWrapper(p)
				}
			}),
		)
		if err != nil {
			return err
		}
		return window.SwitchTo(formWrapper(form))
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
				fields = append(fields, field.New(f.Name, f.String()))
			}
			for _, f := range entry.Details {
				fields = append(fields, field.New(f.Name, f.String()))
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
					return fieldViewClosingPatch{Patch: p}
				})))
			}
		})
	}
}
