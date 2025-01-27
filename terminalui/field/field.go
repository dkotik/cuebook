package field

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/dkotik/cuebook/terminalui/list"
)

type field struct {
	Value    string
	OnSelect tea.Cmd

	width    int
	selected bool
}

func New(name, value string, onSelect tea.Cmd) tea.Model {
	return horizontalLabel{
		Text: name,
		Model: &field{
			Value:    value,
			OnSelect: onSelect,
		},
	}
}

func (f field) Init() (tea.Model, tea.Cmd) {
	return f, nil
}

func (f field) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		f.width = msg.Width
	case list.HighlightHintEvent:
		f.selected = bool(msg)
	case tea.KeyMsg:
		if f.selected {
			switch msg.Key().Code {
			case tea.KeyEnter:
				return f, f.OnSelect
			}
		}
	}
	return f, nil
}
