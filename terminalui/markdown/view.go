package markdown

import (
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

func (m markdown) View() string {
	background := "light"

	if lipgloss.HasDarkBackground() {
		background = "dark"
	}

	r, _ := glamour.NewTermRenderer(
		glamour.WithWordWrap(m.width),
		glamour.WithStandardStyle(background),
	)

	out, _ := r.Render(m.text)
	return out
}
