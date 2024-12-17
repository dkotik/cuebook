package textarea

import (
	"github.com/charmbracelet/bubbles/v2/textarea"
	tea "github.com/charmbracelet/bubbletea/v2"
)

type Textarea struct {
	Label    string
	Required bool

	textarea textarea.Model
}

func (t Textarea) Init() (tea.Model, tea.Cmd) {
	ta := textarea.New()
	ta.Placeholder = "..."
	ta.Focus()

	ta.Prompt = "┃ "
	ta.CharLimit = 280

	ta.SetWidth(30)
	ta.SetHeight(3)

	// Remove cursor line styling
	// ta.FocusedStyle.CursorLine = lipgloss.NewStyle()

	ta.ShowLineNumbers = false

	ta.KeyMap.InsertNewline.SetEnabled(false)
	t.textarea = ta

	return t, nil
}

func (t Textarea) Update(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	t.textarea, cmd = t.textarea.Update(msg)
	return t, nil
}

func (t Textarea) View() string {
	return t.textarea.View()
}
