package list

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type Title struct {
	Text  string
	Style lipgloss.Style
}

func (t Title) Init() tea.Cmd {
	return nil
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

func (t Title) View() tea.View {
	return tea.NewView(t.Style.Render(t.Text))
}
