package entry

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/dkotik/cuebook/terminalui/list"
	"github.com/dkotik/cuebook/terminalui/window"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

var cancelButtonText = &i18n.LocalizeConfig{
	DefaultMessage: &i18n.Message{
		ID:    "bookEntryDiscardUnsavedChanges",
		Other: "Cancel Changes",
	},
}

type cancelEvent struct{}

// func NewcancelButton() tea.Model {
// 	return listForm.NewBlankResponsiveLabel(window.NewTranslatableModel(cancelButton{}))
// }

type cancelButton struct {
	tea.Model
}

func (d cancelButton) Init() (tea.Model, tea.Cmd) {
	return d, nil
}

func (d cancelButton) Translate(lc *i18n.Localizer) (window.TranslatableModel, error) {
	d.Model = list.NewButton(cancelButtonText, func() tea.Msg { return window.BackEvent{} })
	return d, nil
}

func (d cancelButton) Update(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	if d.Model == nil {
		return d, nil
	}
	if _, ok := msg.(cancelEvent); ok {
		d.Model, cmd = deleteButton{}.Init()
		return d.Model, tea.Batch(cmd, tea.RequestWindowSize(), window.RequestLocalizer())
	}
	d.Model, cmd = d.Model.Update(msg)
	return d, cmd
}

func (d cancelButton) View() string {
	if d.Model == nil {
		return "..."
	}
	return d.Model.View()
}
