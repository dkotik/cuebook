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
		PaddingLeft(2)

	if c.Selected {
		style = style.
			BorderLeft(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(c.BorderColor()).
			PaddingLeft(1)
	}

	_, _ = b.WriteString(style.Inherit(styleTitle).Render(c.Title))
	// _, _ = b.WriteRune('\n')

	if len(c.Description) > 0 {
		_, _ = b.WriteRune('\n')
		_, _ = b.WriteString(style.Render(c.Description...))
	}

	return b.String()
}
