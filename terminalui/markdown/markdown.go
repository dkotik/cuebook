package markdown

import (
	tea "github.com/charmbracelet/bubbletea/v2"
)

func New(text string) tea.Model {
	return markdown{text: text}
}

type markdown struct {
	width int
	text  string
	// renderer *glamour.TermRenderer
}

func (m markdown) Init() (tea.Model, tea.Cmd) {
	m.width = 40
	return m, tea.RequestWindowSize()
}

func (m markdown) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	// case markdownRenderer:
	// 	m.text = msg.Render(m.text)
	case tea.WindowSizeMsg:
		m.width = max(10, msg.Width-2)
		// TODO: loading screen
	}
	return m, nil
}
