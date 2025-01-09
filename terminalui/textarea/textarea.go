package textarea

import (
	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/textarea"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/dkotik/cuebook/terminalui/status"
	"github.com/dkotik/cuebook/terminalui/window"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

type Textarea struct {
	Name     string
	Label    string
	Required bool

	status    tea.Model
	saveKey   key.Binding
	escapeKey key.Binding
	textarea  textarea.Model
}

func New(name, label, value string, required bool) tea.Model {
	m := Textarea{
		Name:     name,
		Label:    label,
		Required: required,
		textarea: textarea.New(),
	}
	m.textarea.SetValue(value)
	// ta.Prompt = lipgloss.NewStyle().
	// 	Foreground(lipgloss.Color("243")).
	// 	Render(lipgloss.NormalBorder().Left + " ")
	m.textarea.Prompt = "  "
	m.textarea.Placeholder = "..."
	m.textarea.ShowLineNumbers = false
	// ta.CharLimit = 280
	// ta.KeyMap.InsertNewline.SetEnabled(false)
	// 	// Remove cursor line styling
	// ta.Styles.Focused.CursorLine = lipgloss.NewStyle()
	m.textarea.Styles.Focused.Text = m.textarea.Styles.Focused.Text.Border(lipgloss.NormalBorder(), true, true, true, true)

	lc := i18n.NewLocalizer(i18n.NewBundle(language.AmericanEnglish))
	m.saveKey = window.NewSaveKey(lc)
	m.escapeKey = window.NewCancelKey(lc)
	m.status = status.New(
		window.NewSaveKey(lc),
		window.NewCancelKey(lc),
	)
	return m
}

func (t Textarea) Init() (tea.Model, tea.Cmd) {
	var cmdInitStatus tea.Cmd
	t.status, cmdInitStatus = t.status.Init()
	t.textarea.Focus()
	return t, cmdInitStatus
}

func (t Textarea) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		t.textarea.SetWidth(msg.Width*3/4 - 2)
		t.textarea.SetHeight(msg.Height*3/4 - 2)
	case tea.KeyMsg:
		if key.Matches(msg, t.saveKey) {
			return t, func() tea.Msg {
				return OnChangeEvent{
					TextAreaName: t.Name,
					Value:        t.textarea.Value(),
				}
			}
		}
	}
	var statusCmd, textareaCmd tea.Cmd

	t.status, statusCmd = t.status.Update(msg)
	t.textarea, textareaCmd = t.textarea.Update(msg)
	return t, tea.Batch(statusCmd, textareaCmd)
}
