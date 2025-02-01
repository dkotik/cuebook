package entry

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	listForm "github.com/dkotik/cuebook/terminalui/form"
	"github.com/dkotik/cuebook/terminalui/list"
	"github.com/dkotik/cuebook/terminalui/window"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

var deleteButtonText = &i18n.LocalizeConfig{
	DefaultMessage: &i18n.Message{
		ID:    "bookEntryDelete",
		Other: "Delete Entry",
	},
}

type deleteEvent struct{}

func NewDeleteButton() tea.Model {
	return listForm.NewBlankResponsiveLabel(deleteButton{})
}

type deleteButton struct {
	tea.Model
	Cancel     tea.Model
	ShowCancel bool
}

func (d deleteButton) Init() (tea.Model, tea.Cmd) {
	d.Model = list.NewButton(deleteButtonText, func() tea.Msg {
		return deleteEvent{}
	})
	d.Cancel = NewCancelButton()
	return d, nil
}

func (d deleteButton) Update(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	switch msg.(type) {
	case fieldChangedEvent:
		d.ShowCancel = true
		return d, tea.Batch(window.RequestLocalizer(), tea.RequestWindowSize())
	case cancelEvent:
		d.ShowCancel = false
		return d, tea.Batch(window.RequestLocalizer(), tea.RequestWindowSize())
	}

	if d.ShowCancel {
		d.Cancel, cmd = d.Cancel.Update(msg)
	} else {
		d.Model, cmd = d.Model.Update(msg)
	}
	return d, cmd
}

func (d deleteButton) View() string {
	if d.ShowCancel {
		return d.Cancel.View()
	}
	return d.Model.View()
}
