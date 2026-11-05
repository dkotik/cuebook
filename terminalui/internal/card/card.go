package card

import (
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/dkotik/cuebook/terminalui/internal/list"
)

// Configuration controls card content and appearance. Unset style properties
// inherit the defaults: a bold aqua title and a green left border when active.
// Padding is managed by the card so content stays aligned across states.
type Configuration struct {
	Title            string
	Description      []string
	TitleStyle       lipgloss.Style
	DescriptionStyle lipgloss.Style
	InactiveStyle    lipgloss.Style
	ActiveStyle      lipgloss.Style
}

type card struct {
	Configuration
	Selected bool

	width int
}

func New(c Configuration) tea.Model {
	c.Title = strings.TrimSpace(c.Title)
	if c.Title == "" {
		c.Title = "???"
	}
	// TODO: split description lines by "\n"
	c.Description = slices.DeleteFunc(slices.Clone(c.Description), func(s string) bool {
		return strings.TrimSpace(s) == ""
	})
	c.TitleStyle = c.TitleStyle.Inherit(lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")))
	c.ActiveStyle = c.ActiveStyle.Inherit(lipgloss.NewStyle().
		BorderLeft(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("65")))
	return card{Configuration: c}
}

func (c card) Init() tea.Cmd {
	return nil
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

func (c card) View() tea.View {
	if c.width < 4 {
		return tea.NewView("</>")
	}

	state := c.InactiveStyle
	if c.Selected {
		state = c.ActiveStyle
	}
	style := lipgloss.NewStyle().
		Width(c.width - 3).
		PaddingLeft(max(0, 2-state.GetBorderLeftSize())).
		Inherit(state)

	b := &strings.Builder{}
	_, _ = b.WriteString(style.Inherit(c.TitleStyle).Render(c.Title))
	for _, line := range c.Description {
		_, _ = b.WriteRune('\n')
		_, _ = b.WriteString(style.Inherit(c.DescriptionStyle).Render(line))
	}
	return tea.NewView(b.String())
}
