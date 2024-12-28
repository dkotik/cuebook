package entry

import (
	"github.com/charmbracelet/lipgloss/v2"
)

var (
	inactiveFieldStyle = lipgloss.NewStyle().
				Italic(true).
				Align(lipgloss.Right).
				Foreground(lipgloss.Color("241"))
	activeFieldStyle = lipgloss.NewStyle().Inherit(inactiveFieldStyle).Foreground(lipgloss.Color("43"))

	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("88"))
)

// func (e entry) View() string {
// 	// if len(e.fields) == 0 || e.width < 20 {
// 	// 	return "</>"
// 	// }

// 	// b := strings.Builder{}
// 	// labelWidth := e.width / 4
// 	// valueWidth := e.width - labelWidth - 1

// 	// for _, field := range e.fields {
// 	// 	_, _ = b.WriteString(lipgloss.JoinHorizontal(
// 	// 		lipgloss.Top,
// 	// 		inactiveFieldStyle.
// 	// 			Width(labelWidth).
// 	// 			Render(field.Name+":"),
// 	// 		" ",
// 	// 		valueStyle.
// 	// 			Width(valueWidth).
// 	// 			Render(field.Value),
// 	// 	))
// 	// 	_, _ = b.WriteRune('\n')
// 	// 	_, _ = b.WriteRune('\n')
// 	// }
// 	// return b.String()
// }
