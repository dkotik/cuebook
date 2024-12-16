package card

import (
	"errors"
	"slices"
	"strings"

	"cuelang.org/go/cue"
	tea "github.com/charmbracelet/bubbletea/v2"
)

type card struct {
	Rendered    string
	Title       string
	Description []string
	Selected    bool
}

func (c card) Height() int {
	return len(c.Description) + 1 // for title
}

func (c card) Init() (tea.Model, tea.Cmd) {
	return c, nil
}

func (c card) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		c.Rendered = c.Render(msg)
	}
	return c, nil
}

func New(title string, description ...string) (tea.Model, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, errors.New("empty title")
	}
	// TODO: split description lines by "\n"
	return card{
		Selected: false,
		Title:    title,
		Description: slices.DeleteFunc(description, func(s string) bool {
			return strings.TrimSpace(s) == ""
		}),
	}, nil
}

func NewFromCueStructure(v cue.Value) (tea.Model, error) {
	// field: string @cuebook(detail)

	return New("test")
}
