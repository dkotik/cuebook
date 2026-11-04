package list

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
)

type Title struct {
	Text  string
	Style lipgloss.Style
}

func (t Title) Init() (tea.Model, tea.Cmd) {
	return t, nil
}

func (t Title) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if max := t.Style.GetWidth(); max > msg.Width || max == 0 {
			t.Style = t.Style.Width(msg.Width)
		}
	}
	return t, nil
}

func (t Title) View() string {
	return t.Style.Render(t.Text)
}
