package entry

import (
	"github.com/dkotik/cuebook/terminalui/list"

	tea "github.com/charmbracelet/bubbletea/v2"
)

type Field struct {
	Name  string
	Value string
}

type entry struct {
	list.List

	fields []Field
	width  int
}

func New(fields ...Field) tea.Model {
	return entry{
		fields: fields,
	}
}

// func (e entry) Init() (tea.Model, tea.Cmd) {
// 	return e, nil
// }

// func (e entry) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
// 	switch msg := msg.(type) {
// 	case tea.WindowSizeMsg:
// 		e.width = min(msg.Width, 60)
// 	}
// 	return e, nil
// }
