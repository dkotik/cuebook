package entry

import (
	tea "charm.land/bubbletea/v2"
	"github.com/dkotik/cuebook/terminalui/internal/label"
	"github.com/dkotik/cuebook/terminalui/internal/list"
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
	return label.New(deleteButton{
		Model: list.NewButton(deleteButtonText, func() tea.Msg {
			return deleteEvent{}
		}),
		Cancel: NewCancelButton(),
	}, nil)
}

type deleteButton struct {
	tea.Model
	Cancel     tea.Model
	ShowCancel bool
}

func (d deleteButton) Init() tea.Cmd {
	return tea.Batch(d.Model.Init(), d.Cancel.Init())
}

func (d deleteButton) Update(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	switch msg.(type) {
	case fieldChangedEvent:
		d.ShowCancel = true
		return d, tea.Batch(window.RequestLocalizer(), window.RequestWindowSize())
	case cancelEvent:
		d.ShowCancel = false
		return d, tea.Batch(window.RequestLocalizer(), window.RequestWindowSize())
	}

	if d.ShowCancel {
		d.Cancel, cmd = d.Cancel.Update(msg)
	} else {
		d.Model, cmd = d.Model.Update(msg)
	}
	return d, cmd
}

func (d deleteButton) View() tea.View {
	if d.ShowCancel {
		return d.Cancel.View()
	}
	return d.Model.View()
}
