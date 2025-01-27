package form

import "github.com/charmbracelet/lipgloss/v2"

func (f field) View() string {
	return lipgloss.NewStyle().
		Width(f.width).
		Render(f.Value)
}
