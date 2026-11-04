package textarea

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/textarea"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/dkotik/cuebook/terminalui/window"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

type Command func(string) tea.Cmd

// Configuration controls textarea content and appearance. Label is required.
// A nil OnSubmit does nothing. Unset style properties inherit the defaults: a
// purple frame around a bordered input and an unstyled label.
type Configuration struct {
	Label    string
	Value    string
	OnSubmit Command

	LabelStyle lipgloss.Style
	FrameStyle lipgloss.Style
	TextStyle  lipgloss.Style
}

type Textarea struct {
	Configuration

	saveKey  key.Binding
	textarea textarea.Model
}

func New(c Configuration) (_ tea.Model, err error) {
	if strings.TrimSpace(c.Label) == "" {
		return nil, fmt.Errorf("invalid textarea configuration: %w", errors.New("label is required"))
	}
	if c.OnSubmit == nil {
		c.OnSubmit = func(string) tea.Cmd { return nil }
	}
	c.FrameStyle = c.FrameStyle.Inherit(lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("55")))

	m := Textarea{
		Configuration: c,
		textarea:      textarea.New(),
	}
	if c.Value != "" {
		m.textarea.SetValue(c.Value)
	}
	m.textarea.Prompt = "  "
	m.textarea.Placeholder = "..."
	m.textarea.ShowLineNumbers = false
	m.textarea.CharLimit = 1 << 20 // 1mb
	focused := m.textarea.Styles.Focused.Text.Border(lipgloss.NormalBorder(), true, true, true, true)
	m.textarea.Styles.Focused.Text = c.TextStyle.Inherit(focused)

	lc := i18n.NewLocalizer(i18n.NewBundle(language.AmericanEnglish))
	m.saveKey = window.NewSaveKey(lc)
	return m, nil
}

func (t Textarea) Init() (tea.Model, tea.Cmd) {
	t.textarea.Focus()
	t.textarea = ScrollFix(t.textarea) // TODO: does not seem to work
	return t, tea.RequestWindowSize()
}

func (t Textarea) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		t.textarea.SetWidth(msg.Width*3/4 - 2)
		t.textarea.SetHeight(msg.Height*3/4 - 2)
	case tea.KeyMsg:
		if key.Matches(msg, t.saveKey) {
			return t, t.OnSubmit(t.textarea.Value())
		}
	}
	var textareaCmd tea.Cmd
	t.textarea, textareaCmd = t.textarea.Update(msg)
	return t, textareaCmd
}

func (t Textarea) View() string {
	return lipgloss.JoinVertical(lipgloss.Left,
		t.LabelStyle.Render(" "+t.Label+":"),
		t.FrameStyle.Render(t.textarea.View()))
}

func (t Textarea) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("label", t.Label),
		slog.String("value", t.textarea.Value()),
	)
}
