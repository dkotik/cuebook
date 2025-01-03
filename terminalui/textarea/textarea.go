package textarea

import (
	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/textarea"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/dkotik/cuebook/terminalui"
	"github.com/dkotik/cuebook/terminalui/status"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

type Textarea struct {
	Label    string
	Required bool

	status    tea.Model
	saveKey   key.Binding
	escapeKey key.Binding
	textarea  textarea.Model
}

func (t Textarea) Init() (tea.Model, tea.Cmd) {
	lc := i18n.NewLocalizer(i18n.NewBundle(language.AmericanEnglish))
	t.saveKey = terminalui.NewSaveKey(lc)
	t.escapeKey = terminalui.NewCancelKey(lc)

	ta := textarea.New()
	ta.Placeholder = "..."
	ta.Focus()

	ta.Prompt = "  "
	// ta.Prompt = lipgloss.NewStyle().
	// 	Foreground(lipgloss.Color("243")).
	// 	Render(lipgloss.NormalBorder().Left + " ")
	ta.CharLimit = 280

	ta.SetWidth(46)
	ta.SetHeight(6)

	// Remove cursor line styling
	// ta.Styles.Focused.CursorLine = lipgloss.NewStyle()
	ta.Styles.Focused.Text = ta.Styles.Focused.Text.Border(lipgloss.NormalBorder(), true, true, true, true)

	ta.ShowLineNumbers = false

	// ta.KeyMap.InsertNewline.SetEnabled(false)
	t.textarea = ta
	t.status = status.New(
		terminalui.NewSaveKey(lc),
		terminalui.NewCancelKey(lc),
	)
	var cmdInitStatus tea.Cmd
	t.status, cmdInitStatus = t.status.Init()
	// t.textarea, cmdInitTextarea = t.textarea.Init()
	return t, cmdInitStatus
}

func (t Textarea) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// if t.status == nil {
	// 	return t, nil
	// }

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, t.saveKey) {
			return t, func() tea.Msg {
				return OnChangeEvent(t.textarea.Value())
			}
		}
	}
	var statusCmd, textareaCmd tea.Cmd

	t.status, statusCmd = t.status.Update(msg)
	t.textarea, textareaCmd = t.textarea.Update(msg)
	return t, tea.Batch(statusCmd, textareaCmd)
}
