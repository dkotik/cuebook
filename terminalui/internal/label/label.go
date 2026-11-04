package label

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/dkotik/cuebook/terminalui/list"
)

// DefaultInactiveColor is the foreground color for unselected labels.
const DefaultInactiveColor = "#626468"

// LayoutChangeEvent aligns labels beside their models or above them.
// Width is the label column width when IsHorizontal is true.
type LayoutChangeEvent struct {
	Width        int
	IsHorizontal bool
}

// Configuration controls label content and appearance. Unset style properties
// inherit the defaults: gray inactive text, blue active text, and a red marker.
type Configuration struct {
	Text                string
	Required            bool
	InactiveStyle       lipgloss.Style
	ActiveStyle         lipgloss.Style
	RequiredMarkerStyle lipgloss.Style
}

func (c Configuration) Vertical() Configuration {
	c.InactiveStyle = c.InactiveStyle.Align(lipgloss.Left).PaddingRight(0).UnsetWidth()
	c.ActiveStyle = c.ActiveStyle.Align(lipgloss.Left).PaddingRight(0).UnsetWidth()
	return c
}

func (c Configuration) Horizontal(withWidth int) Configuration {
	c.InactiveStyle = c.InactiveStyle.Align(lipgloss.Right).PaddingRight(1).Width(withWidth)
	c.ActiveStyle = c.ActiveStyle.Align(lipgloss.Right).PaddingRight(1).Width(withWidth)
	return c
}

// New wraps a model in a responsive label. A nil configuration creates a blank
// label using the default styles.
func New(wrap tea.Model, configuration *Configuration) tea.Model {
	var c Configuration
	if configuration != nil {
		c = *configuration
	}
	c.InactiveStyle = c.InactiveStyle.Inherit(lipgloss.NewStyle().Foreground(lipgloss.Color(DefaultInactiveColor)))
	c.ActiveStyle = c.ActiveStyle.Inherit(lipgloss.NewStyle().Foreground(lipgloss.Blue))
	c.RequiredMarkerStyle = c.RequiredMarkerStyle.Inherit(lipgloss.NewStyle().Foreground(lipgloss.BrightRed))
	return horizontalLabel{Model: wrap, Configuration: c}
}

type horizontalLabel struct {
	tea.Model
	Configuration
	Focused bool
}

func (l horizontalLabel) Init() (_ tea.Model, cmd tea.Cmd) {
	l.Model, cmd = l.Model.Init()
	return l, cmd
}

func (l horizontalLabel) Update(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	switch msg := msg.(type) {
	case list.HighlightHintEvent:
		l.Focused = bool(msg)
	case LayoutChangeEvent:
		if !msg.IsHorizontal {
			l.Configuration = l.Configuration.Vertical()
			return verticalLabel{l}, nil
		}
		l.Configuration = l.Configuration.Horizontal(msg.Width)
		return l, nil
	case tea.WindowSizeMsg:
		msg.Width -= l.GetStyle().GetWidth()
		l.Model, cmd = l.Model.Update(msg)
		return l, cmd
	}
	l.Model, cmd = l.Model.Update(msg)
	return l, cmd
}

func (l horizontalLabel) GetStyle() lipgloss.Style {
	if l.Focused {
		return l.ActiveStyle
	}
	return l.InactiveStyle
}

func (l horizontalLabel) Render() string {
	if l.Text == "" {
		return l.InactiveStyle.Render("")
	}
	if l.Required {
		return l.RequiredMarkerStyle.Render("✱") + l.GetStyle().Render(l.Text+":")
	}
	return l.GetStyle().Render(l.Text + ":")
}

func (l horizontalLabel) View() string {
	return lipgloss.JoinHorizontal(
		lipgloss.Left,
		l.Render(),
		l.Model.View(),
	)
}

type verticalLabel struct {
	horizontalLabel
}

func (l verticalLabel) Update(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	switch msg := msg.(type) {
	case list.HighlightHintEvent:
		l.Focused = bool(msg)
	case LayoutChangeEvent:
		if msg.IsHorizontal {
			l.Configuration = l.Configuration.Horizontal(msg.Width)
			return l.horizontalLabel, nil
		}
		l.Configuration = l.Configuration.Vertical()
		return l, nil
	}
	l.Model, cmd = l.Model.Update(msg)
	return l, cmd
}

func (l verticalLabel) View() string {
	if l.Text == "" {
		return l.Model.View()
	}
	return lipgloss.JoinVertical(
		lipgloss.Left,
		l.Render(),
		l.Model.View(),
	)
}
