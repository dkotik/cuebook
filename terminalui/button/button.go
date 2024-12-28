package button

import (
	tea "github.com/charmbracelet/bubbletea/v2"
)

type button struct {
	tea.Cmd

	viewport tea.WindowSizeMsg
	shortcut tea.Key
}

func New(text string, shortcut tea.Key, onClick tea.Cmd) tea.Model {
	return button{
		shortcut: shortcut,
		Cmd:      onClick,
	}
}

func (b button) Init() (tea.Model, tea.Cmd) {
	return b, nil
}

func (b button) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		b.viewport = msg
	case tea.MouseMotionMsg:
	}
	return b, nil
}

func (b button) View() string {
	return "button"
}
