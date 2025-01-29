package form

import (
	"github.com/charmbracelet/bubbles/v2/textarea"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/dkotik/cuebook/terminalui/list"
	ta "github.com/dkotik/cuebook/terminalui/textarea"
)

type field struct {
	Value    string
	Input    textarea.Model
	OnChange func(string) tea.Cmd

	width    int
	selected bool
}

func NewField(name, value string, onChange func(string) tea.Cmd) tea.Model {
	input := textarea.New()
	input.SetValue(value)
	// input.SetHeight(max(2, min(lipgloss.Height(value), 6)))
	return horizontalLabel{
		Text: name,
		Model: field{
			Value:    value,
			Input:    input,
			OnChange: onChange,
		},
	}
}

func (f field) Init() (tea.Model, tea.Cmd) {
	return f, nil
}

func (f field) Focus() (tea.Model, tea.Cmd) {
	f.selected = true
	height := lipgloss.Height(f.Value)
	f.Input.SetHeight(max(2, min(height, 6)))
	f.Input.Focus()
	f.Input = ta.ScrollFix(f.Input)
	return f, nil
}

func (f field) Blur() (tea.Model, tea.Cmd) {
	f.selected = false
	// f.Input.Blur()
	return f, nil
}

func (f field) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		f.width = msg.Width
		f.Input.SetWidth(f.width)
	case list.HighlightHintEvent:
		if msg {
			return f.Focus()
		} else {
			return f.Blur()
		}
	case tea.KeyMsg:
		if f.selected {
			var cmd tea.Cmd
			f.Input, cmd = f.Input.Update(msg)

			if cmd == nil {
				switch msg.Key().Code {
				case tea.KeyEnter:
				case tea.KeyEscape:
					f.selected = false
					f.Input.Blur()
				}
			} else {
				f.Value = f.Input.Value()
				// f.Input.SetHeight(min(lipgloss.Height(f.Value), 6))
				// f.Input, _ = f.Input.Update(nil) // to reposition view
			}

			return f, tea.Batch(cmd, f.OnChange(f.Value))
		} else if msg.Key().Code == tea.KeyEnter {
			return f.Focus()
		}
	}
	return f, nil
}

func (f field) View() string {
	if f.selected {
		// f.Input.Focus()
		return f.Input.View()
	}

	return lipgloss.NewStyle().
		Width(f.width).
		Render(f.Value)
}
