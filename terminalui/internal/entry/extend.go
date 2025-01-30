package entry

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	listForm "github.com/dkotik/cuebook/terminalui/form"
	"github.com/dkotik/cuebook/terminalui/list"
	"github.com/dkotik/cuebook/terminalui/textarea"
	"github.com/dkotik/cuebook/terminalui/window"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

var extendButtonText = &i18n.LocalizeConfig{
	DefaultMessage: &i18n.Message{
		ID:    "bookEntryExtend",
		Other: "Add a Custom Field",
	},
}

type extendEvent struct {
	Name string
}

func NewExtendButton() tea.Model {
	return listForm.NewBlankResponsiveLabel(window.NewTranslatableModel(extendButton{}))
}

type extendButton struct {
	tea.Model
}

func (d extendButton) Init() (tea.Model, tea.Cmd) {
	return d, nil
}

func (d extendButton) Translate(lc *i18n.Localizer) (window.TranslatableModel, error) {
	text, err := lc.Localize(extendButtonText)
	if err != nil {
		return nil, err
	}
	d.Model = list.NewButton(text, func() tea.Msg {
		ta, err := textarea.New(
			textarea.WithLabel(text),
			textarea.WithOnSubmitCommand(func(value string) tea.Cmd {
				return func() tea.Msg {
					return extendEvent{
						Name: value,
					}
				}
			}),
		)
		if err != nil {
			return err
		}
		return window.SwitchTo(ta)
	})
	return d, nil
}

func (d extendButton) Update(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	if d.Model == nil {
		return d, nil
	}
	d.Model, cmd = d.Model.Update(msg)
	return d, cmd
}

func (d extendButton) View() string {
	if d.Model == nil {
		return "..."
	}
	return d.Model.View()
}
