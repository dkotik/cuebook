package button

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/dkotik/cuebook/terminalui/internal/list"
)

// Configuration controls button content and appearance. Unset style properties
// inherit the defaults: gray inactive text and teal active text, matching the
// list buttons.
type Configuration struct {
	Text          string
	Shortcut      tea.Key
	OnClick       tea.Cmd
	InactiveStyle lipgloss.Style
	ActiveStyle   lipgloss.Style
}

type button struct {
	Configuration
	Selected bool

	viewport tea.WindowSizeMsg
}

func New(c Configuration) tea.Model {
	c.InactiveStyle = c.InactiveStyle.Inherit(lipgloss.NewStyle().Foreground(lipgloss.Color("#999999")))
	c.ActiveStyle = c.ActiveStyle.Inherit(lipgloss.NewStyle().Foreground(lipgloss.Color("73")))
	return button{Configuration: c}
}

func (b button) Init() (tea.Model, tea.Cmd) {
	return b, nil
}

func (b button) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		b.viewport = msg
	case list.HighlightHintEvent:
		b.Selected = bool(msg)
	}
	return b, nil
}

func (b button) View() string {
	if b.Selected {
		return b.ActiveStyle.Render(b.Text)
	}
	return b.InactiveStyle.Render(b.Text)
}
