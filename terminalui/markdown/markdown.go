package markdown

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/dkotik/cuebook/terminalui/middleware"
)

func New(text string) tea.Model {
	return middleware.NewCache(text).Wrap(markdown{text: text})
	// return markdown{text: text}
}

type markdown struct {
	width int
	text  string
}

func (m markdown) Init() (tea.Model, tea.Cmd) {
	m.width = 40
	return m, tea.RequestWindowSize()
}

func (m markdown) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = max(10, msg.Width-2)
		// TODO: loading screen
	}
	return m, nil
}
