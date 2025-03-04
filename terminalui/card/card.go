package card

import (
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/dkotik/cuebook/terminalui/list"
)

type card struct {
	ID          string
	Title       string
	Description []string
	Selected    bool

	width int
}

func (c card) Init() (tea.Model, tea.Cmd) {
	return c, nil
}

func (c card) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		c.width = msg.Width
	case list.HighlightHintEvent:
		c.Selected = bool(msg)
	}
	return c, nil
}

func New(title string, description ...string) tea.Model {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "???"
	}
	// TODO: split description lines by "\n"
	return card{
		Selected: false,
		Title:    title,
		Description: slices.DeleteFunc(description, func(s string) bool {
			return strings.TrimSpace(s) == ""
		}),
	}
}
