package list

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func NewButton(text *i18n.LocalizeConfig, command tea.Cmd) tea.Model {
	if text == nil {
		panic("text is empty")
	}
	temp := "..."
	if text.DefaultMessage != nil {
		temp = "`" + text.DefaultMessage.Other + "`"
	}
	return button{
		Text:         temp,
		Command:      command,
		TextTemplate: text,
	}
}

type button struct {
	Text         string
	TextTemplate *i18n.LocalizeConfig
	Command      tea.Cmd
	Width        int
	Focused      bool
}

func (b button) Init() (tea.Model, tea.Cmd) {
	return b, nil
}

func (b button) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case *i18n.Localizer:
		// panic("..")
		text, err := msg.Localize(b.TextTemplate)
		if err != nil {
			return b, func() tea.Msg { return err }
		}
		b.Text = text
	case tea.WindowSizeMsg:
		b.Width = msg.Width / 2
	case HighlightHintEvent:
		b.Focused = bool(msg)
	case tea.KeyMsg:
		if msg.Key().Code == tea.KeyEnter {
			return b, b.Command
		}
	}
	return b, nil
}

func (b button) View() string {
	style := lipgloss.NewStyle().
		Width(b.Width).
		Foreground(lipgloss.Color("#999999"))
		// Padding(1).
		// Background(lipgloss.Color("#333333"))
		// PaddingLeft(2)

	if b.Focused {
		style = style.
			Foreground(lipgloss.Color("73"))
		// BorderLeft(true).
		// BorderStyle(lipgloss.NormalBorder()).
		// BorderForeground(lipgloss.Color("65"))
		// Background(lipgloss.Color("#3399AA"))
		// PaddingLeft(1)
	}

	return style.Render("● " + b.Text)
}
