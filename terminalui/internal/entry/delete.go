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
	return listForm.NewBlankResponsiveLabel(window.NewTranslatableModel(deleteButton{}))
}

type deleteButton struct {
	tea.Model
}

func (d deleteButton) Init() (tea.Model, tea.Cmd) {
	return d, nil
}

func (d deleteButton) Translate(lc *i18n.Localizer) (window.TranslatableModel, error) {
	text, err := lc.Localize(deleteButtonText)
	if err != nil {
		return nil, err
	}
	d.Model = list.NewButton(text, func() tea.Msg { return deleteEvent{} })
	return d, nil
}

func (d deleteButton) Update(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	if d.Model == nil {
		return d, nil
	}
	if _, ok := msg.(fieldChangedEvent); ok {
		d.Model, cmd = cancelButton{}.Init()
		return d.Model, tea.Batch(cmd, tea.RequestWindowSize(), window.RequestLocalizer())
	}
	d.Model, cmd = d.Model.Update(msg)
	return d, cmd
}

func (d deleteButton) View() string {
	if d.Model == nil {
		return "..."
	}
	return d.Model.View()
}
