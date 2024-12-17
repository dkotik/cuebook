package textarea

import (
	"github.com/charmbracelet/bubbles/v2/help"
	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/textarea"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/dkotik/cuebook/terminalui"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

type Textarea struct {
	Label    string
	Required bool
	Save     func(string) tea.Cmd

	saveKey   key.Binding
	escapeKey key.Binding
	textarea  textarea.Model
	help      help.Model
}

func (t Textarea) Init() (tea.Model, tea.Cmd) {
	if t.Label == "" {
		t.Label = "???"
	}
	if t.Save == nil {
		t.Save = func(_ string) tea.Cmd {
			return func() tea.Msg {
				return terminalui.Back{}
			}
		}
	}

	lc := i18n.NewLocalizer(i18n.NewBundle(language.AmericanEnglish))
	t.saveKey = terminalui.NewSaveKey(lc)
	t.escapeKey = terminalui.NewCancelKey(lc)

	ta := textarea.New()
	ta.Placeholder = "..."
	ta.Focus()

	ta.Prompt = ""
	// ta.Prompt = lipgloss.NewStyle().
	// 	Foreground(lipgloss.Color("243")).
	// 	Render(lipgloss.NormalBorder().Left + " ")
	ta.CharLimit = 280

	ta.SetWidth(40)
	ta.SetHeight(6)

	// Remove cursor line styling
	// ta.Styles.Focused.CursorLine = lipgloss.NewStyle()

	ta.ShowLineNumbers = false

	// ta.KeyMap.InsertNewline.SetEnabled(false)
	t.textarea = ta
	t.help = help.New()

	return t, nil
}

func (t Textarea) Update(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, t.saveKey) {
			return t, t.Save(t.textarea.Value())
		}
	}
	t.textarea, cmd = t.textarea.Update(msg)
	return t, nil
}
