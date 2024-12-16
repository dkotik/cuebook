package card

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
	"github.com/dkotik/cuebook/terminalui"
	"github.com/muesli/reflow/truncate"
)

var (
	styleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: "236", Dark: "86"}) // aqua

	borderInactive = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Render(" " + lipgloss.ThickBorder().Left)
	borderActive = lipgloss.NewStyle().
			Foreground(lipgloss.Color("65")).
			Render(" " + lipgloss.ThickBorder().Left)
)

func (c card) Render(size tea.WindowSizeMsg) string {
	width := size.Width
	if width < 4 {
		return strings.Repeat(" ", width)
	}
	b := &strings.Builder{}

	var border string
	if c.Selected {
		border = borderActive
	} else {
		border = borderInactive
	}
	_, _ = b.WriteString(border)
	width -= 3

	if l := len(c.Title); l == 0 {
		_, _ = b.WriteString(strings.Repeat("?", width))
	} else {
		_, _ = b.WriteString(
			styleTitle.Render(
				terminalui.PadLine(c.Title, width-1),
			),
		)
		// _, _ = b.WriteString(strings.Repeat("-", width-l-1))
		// _, _ = b.WriteRune('\n')
	}

	for _, line := range c.Description {
		_, _ = b.WriteRune('\n')
		_, _ = b.WriteString(border)
		line = truncate.String(line, uint(width))
		_, _ = b.WriteString(line)
		_, _ = b.WriteString(strings.Repeat(".", width-len(line)-1))
	}

	return b.String()
}

func (c card) View() string {
	return c.Rendered
}
