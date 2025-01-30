package entry

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	listForm "github.com/dkotik/cuebook/terminalui/form"
	"github.com/dkotik/cuebook/terminalui/list"
	"github.com/dkotik/cuebook/terminalui/window"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

var saveButtonText = &i18n.LocalizeConfig{
	DefaultMessage: &i18n.Message{
		ID:    "bookEntrySave",
		Other: "Save Changes",
	},
}

func NewSaveButton() tea.Model {
	return listForm.NewBlankResponsiveLabel(window.NewTranslatableModel(saveButton{}))
}

type saveButton struct {
	tea.Model
}

func (s saveButton) Init() (tea.Model, tea.Cmd) {
	return s, nil
}

func (s saveButton) Translate(lc *i18n.Localizer) (window.TranslatableModel, error) {
	text, err := lc.Localize(saveButtonText)
	if err != nil {
		return nil, err
	}
	s.Model = list.NewButton(text, func() tea.Msg { return listForm.SaveChangesEvent{} })
	return s, nil
}

func (s saveButton) Update(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	if s.Model == nil {
		return s, nil
	}
	s.Model, cmd = s.Model.Update(msg)
	return s, cmd
}

func (s saveButton) View() string {
	if s.Model == nil {
		return "..."
	}
	return s.Model.View()
}
