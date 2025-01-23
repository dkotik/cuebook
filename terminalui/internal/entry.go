package internal

import (
	tea "github.com/charmbracelet/bubbletea/v2"
)

type entrySelected int

type Entry struct {
	tea.Model
	Index int
}

func (e Entry) Init() (_ tea.Model, cmd tea.Cmd) {
	e.Model, cmd = e.Model.Init()
	return e, cmd
}

func (e Entry) Update(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Key().Code {
		case tea.KeyEnter:
			return e, func() tea.Msg { return entrySelected(e.Index) }
		}
	}
	e.Model, cmd = e.Model.Update(msg)
	return e, cmd
}
