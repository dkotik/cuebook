package card

import (
	"image/color"
	"strings"

	"github.com/charmbracelet/lipgloss/v2"
)

var (
	styleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(86)) // aqua
	// lipgloss.AdaptiveColor{Light: "236", Dark: "86"}

	borderInactiveColor = lipgloss.Color("235")
	borderActiveColor   = lipgloss.Color("65")
	// borderInactive = lipgloss.NewStyle().
	// 		Foreground(lipgloss.Color("243")).
	// 		Render(" " + lipgloss.ThickBorder().Left)
	// borderActive = lipgloss.NewStyle().
	// 		Foreground(lipgloss.Color("65")).
	// 		Render(" " + lipgloss.ThickBorder().Left)
)

// func (c card) View() string {
// width := size.Width
// if width < 4 {
// 	return strings.Repeat(" ", width)
// }
// b := &strings.Builder{}

// var border string
// if c.Selected {
// 	border = borderActive
// } else {
// 	border = borderInactive
// }
// _, _ = b.WriteString(border)
// width -= 3

// if l := len(c.Title); l == 0 {
// 	_, _ = b.WriteString(strings.Repeat("?", width))
// } else {
// 	_, _ = b.WriteString(
// 		styleTitle.Render(
// 			terminalui.PadLine(c.Title, width-1),
// 		),
// 	)
// 	// _, _ = b.WriteString(strings.Repeat("-", width-l-1))
// 	// _, _ = b.WriteRune('\n')
// }

// for _, line := range c.Description {
// 	_, _ = b.WriteRune('\n')
// 	_, _ = b.WriteString(border)
// 	line = truncate.String(line, uint(width))
// 	_, _ = b.WriteString(line)
// 	// _, _ = b.WriteString(strings.Repeat(".", width-len(line)-1))
// }

// return b.String()
// }

func (c card) BorderColor() color.Color {
	if c.Selected {
		return borderActiveColor
	}
	return borderInactiveColor
}

func (c card) View() string {
	if c.width < 4 {
		return "</>"
	}
	width := c.width - 3

	b := &strings.Builder{}
	style := lipgloss.NewStyle().
		Width(width).
		BorderLeft(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(c.BorderColor()).
		PaddingLeft(1)

	_, _ = b.WriteString(style.Inherit(styleTitle).Render(c.Title))
	// _, _ = b.WriteRune('\n')

	if len(c.Description) > 0 {
		_, _ = b.WriteRune('\n')
		_, _ = b.WriteString(style.Render(c.Description...))
	}

	return b.String()
}
