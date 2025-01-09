package terminalui

import (
	tea "github.com/charmbracelet/bubbletea/v2"
)

type eventAdaptor[T any] struct {
	tea.Model
	update func(tea.Model, T) (tea.Model, tea.Cmd)
}

func (u eventAdaptor[T]) Init() (_ tea.Model, cmd tea.Cmd) {
	u.Model, cmd = u.Model.Init()
	return u, cmd
}

func (u eventAdaptor[T]) Update(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	switch msg := msg.(type) {
	case T:
		u.Model, cmd = u.update(u.Model, msg)
	default:
		u.Model, cmd = u.Model.Update(msg)
	}
	return u, cmd
}

func NewEventAdaptor[T any](
	update func(tea.Model, T) (tea.Model, tea.Cmd),
) func(next tea.Model) tea.Model {
	if update == nil {
		panic("update call is nil")
	}
	return func(next tea.Model) tea.Model {
		if next == nil {
			panic("wrapped model is nil")
		}
		return eventAdaptor[T]{
			Model:  next,
			update: update,
		}
	}
}

func NewKeySwitchAdaptor(
	key tea.Key,
	constructor func() tea.Model,
) func(tea.Model) tea.Model {
	return NewEventAdaptor(func(m tea.Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
		// if msg.Key() == key {
		// 	return m, func() tea.Msg {
		// 		return SwitchTo(constructor())
		// 	}
		// }
		return m.Update(msg)
	})
}
