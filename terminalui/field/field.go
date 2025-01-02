package field

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/dkotik/cuebook/terminalui/list"
)

type field struct {
	Name  string
	Value string

	labelWidth int
	valueWidth int
	selected   bool
}

func New(name, value string) tea.Model {
	return &field{
		Name:  name,
		Value: value,
	}
}

func (f field) Init() (tea.Model, tea.Cmd) {
	return f, nil
}

func (f field) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		f.labelWidth = msg.Width / 4
		f.valueWidth = msg.Width - f.labelWidth
	case list.SelectionActivateEvent:
		f.selected = bool(msg)
	}
	return f, nil
}
