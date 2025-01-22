package textarea

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/textarea"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/dkotik/cuebook/terminalui/window"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

type Textarea struct {
	Name     string
	Label    string
	OnSubmit Command
	// Required bool

	saveKey key.Binding
	// escapeKey key.Binding
	textarea textarea.Model
}

func New(withOptions ...Option) (_ tea.Model, err error) {
	o := &options{}
	for _, option := range append(withOptions, func(o *options) error {
		if o.OnSubmit == nil {
			o.OnSubmit = func(string) tea.Cmd { return nil }
		}
		if o.Label == "" {
			return errors.New("label is required")
		}
		return nil
	}) {
		if err = option(o); err != nil {
			return nil, fmt.Errorf("invalid textarea options: %w", err)
		}
	}

	m := Textarea{
		Label:    o.Label,
		OnSubmit: o.OnSubmit,
		// Required: required,
		textarea: textarea.New(),
	}
	if o.Value != "" {
		m.textarea.SetValue(o.Value)
	}
	// ta.Prompt = lipgloss.NewStyle().
	// 	Foreground(lipgloss.Color("243")).
	// 	Render(lipgloss.NormalBorder().Left + " ")
	m.textarea.Prompt = "  "
	m.textarea.Placeholder = "..."
	m.textarea.ShowLineNumbers = false
	m.textarea.CharLimit = 1 << 20 // 1mb
	// ta.KeyMap.InsertNewline.SetEnabled(false)
	// 	// Remove cursor line styling
	// ta.Styles.Focused.CursorLine = lipgloss.NewStyle()
	m.textarea.Styles.Focused.Text = m.textarea.Styles.Focused.Text.Border(lipgloss.NormalBorder(), true, true, true, true)

	lc := i18n.NewLocalizer(i18n.NewBundle(language.AmericanEnglish))
	m.saveKey = window.NewSaveKey(lc)
	// m.escapeKey = window.NewCancelKey(lc)
	return m, nil
}

func (t Textarea) Init() (tea.Model, tea.Cmd) {
	t.textarea.Focus()
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
