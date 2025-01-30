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
		var cmd tea.Cmd
		switch msg.Key().Code {
		case tea.KeyUp:
			if f.Input.Line() == 0 {
				return f, nil
			}
		case tea.KeyLeft:
			if f.Input.Line() == 0 && f.Input.LineInfo().ColumnOffset == 0 {
				return f, nil
			}
		case tea.KeyDown:
			if f.Input.Line() == f.Input.LineCount()-1 {
				return f, nil
			}
		case tea.KeyRight:
			if line := f.Input.LineInfo(); f.Input.Line() == f.Input.LineCount()-1 && line.ColumnOffset == line.Width-1 {
				// info := f.Input.LineInfo()
				return f, nil
			}
		}
		f.Input, cmd = f.Input.Update(msg)
		f.Value = f.Input.Value()
		return f, tea.Batch(cmd, f.OnChange(f.Value))
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
