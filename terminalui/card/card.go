package card

import (
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea/v2"
)

type Card struct {
	ID          string
	Title       string
	Description []string
	Selected    bool

	rendered string
}

func (c Card) Height() int {
	return len(c.Description) + 1 // for title
}

func (c Card) Init() (tea.Model, tea.Cmd) {
	return c, nil
}

func (c Card) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		c.rendered = c.Render(msg)
	}
	return c, nil
}

func New(title string, description ...string) (Card, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "???"
	}
	// TODO: split description lines by "\n"
	return Card{
		Selected: false,
		Title:    title,
		Description: slices.DeleteFunc(description, func(s string) bool {
			return strings.TrimSpace(s) == ""
		}),
	}, nil
}
