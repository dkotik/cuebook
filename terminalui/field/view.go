package field

import "github.com/charmbracelet/lipgloss/v2"

func (f field) View() string {
	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		f.Label(),
		lipgloss.NewStyle().
			Width(f.valueWidth*3/4).
			Render(f.Value),
	)
}

func (f field) Label() string {
	style := lipgloss.NewStyle().
		Width(f.labelWidth).
		PaddingRight(1).
		AlignHorizontal(lipgloss.Right).
		// AlignVertical(lipgloss.Top).
		Foreground(lipgloss.Color("241"))

	if f.selected {
		style = style.Foreground(lipgloss.Color("43"))
	}
	return style.Render(f.Name + ":")
}
