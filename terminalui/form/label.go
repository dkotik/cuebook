package form

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/dkotik/cuebook/terminalui/list"
)

type labelStyleChangeEvent struct {
	Style        LabelStyle
	IsHorizontal bool
}

type LabelStyle struct {
	Blurred        lipgloss.Style
	Focused        lipgloss.Style
	RequiredMarker lipgloss.Style
	// Modified lipgloss.Style
}

func (s LabelStyle) Vertical() LabelStyle {
	return LabelStyle{
		Blurred:        s.Blurred.Align(lipgloss.Left).PaddingRight(0).UnsetWidth(),
		Focused:        s.Focused.Align(lipgloss.Left).PaddingRight(0).UnsetWidth(),
		RequiredMarker: s.RequiredMarker,
	}
}

func (s LabelStyle) Horizontal(withWidth int) LabelStyle {
	return LabelStyle{
		Blurred:        s.Blurred.Align(lipgloss.Right).PaddingRight(1).Width(withWidth),
		Focused:        s.Focused.Align(lipgloss.Right).PaddingRight(1).Width(withWidth),
		RequiredMarker: s.RequiredMarker,
	}
}

func NewBlankResponsiveLabel(wrap tea.Model) tea.Model {
	return horizontalLabel{Model: wrap}
}

type horizontalLabel struct {
	tea.Model
	Text     string
	Style    LabelStyle
	Focused  bool
	required bool
	// Modified bool
}

func (l horizontalLabel) Init() (_ tea.Model, cmd tea.Cmd) {
	l.Model, cmd = l.Model.Init()
	return l, cmd
}

func (l horizontalLabel) Update(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	switch msg := msg.(type) {
	case list.HighlightHintEvent:
		l.Focused = bool(msg)
	case labelStyleChangeEvent:
		l.Style = msg.Style
		if !msg.IsHorizontal {
			return verticalLabel{l}, nil
		}
		return l, nil
	case tea.WindowSizeMsg:
		msg.Width = msg.Width - l.GetStyle().GetWidth()
	}
	l.Model, cmd = l.Model.Update(msg)
	return l, cmd
}

func (l horizontalLabel) GetStyle() lipgloss.Style {
	if l.Focused {
		return l.Style.Focused
	}
	return l.Style.Blurred
}

func (l horizontalLabel) Render() string {
	if l.Text == "" {
		return l.Style.Blurred.Render("")
	}
	if l.required {
		return l.Style.RequiredMarker.Render("✱") + l.GetStyle().Render(l.Text+":")
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
	case labelStyleChangeEvent:
		l.Style = msg.Style
		if msg.IsHorizontal {
			return l.horizontalLabel, nil
		}
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
