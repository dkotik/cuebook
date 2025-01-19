package markdown

import (
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

var background = "dark"

func init() {
	// TODO: this made Markdown rendering slow
	// ought to be moved into style package?
	if !lipgloss.HasDarkBackground() {
		background = "light"
	}
}

func (m markdown) View() string {
	r, _ := glamour.NewTermRenderer(
		glamour.WithWordWrap(m.width),
		glamour.WithStandardStyle(background),
	)

	out, _ := r.Render(m.text)
	return out
}
