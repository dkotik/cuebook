package terminalui

import tea "github.com/charmbracelet/bubbletea/v2"

func NewCached(model tea.Model) tea.Model {
	if model == nil {
		panic("model is nil")
	}
	return cachedModel{Model: model}
}

type cachedModel struct {
	tea.Model

	view string
}

func (c cachedModel) Init() (model tea.Model, cmd tea.Cmd) {
	model, cmd = c.Model.Init()
	return cachedModel{Model: model}, cmd
}

func (c cachedModel) Update(msg tea.Msg) (model tea.Model, cmd tea.Cmd) {
	model, cmd = c.Model.Update(msg)
	return cachedModel{Model: model}, cmd
}

func (c cachedModel) View() string {
	if c.view == "" {
		c.view = c.Model.View()
	}
	return c.view
}
