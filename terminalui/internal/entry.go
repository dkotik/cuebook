package internal

import (
	tea "github.com/charmbracelet/bubbletea/v2"
)

type entrySelected int

// type entrySwap struct {
// 	SelectedIndex int
// 	TargetIndex   int
// }

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
		// case 'k', tea.KeyUp:
		// 	if e.Index > 0 && !msg.Key().Mod.Contains(tea.ModCtrl) {
		// 		break
		// 	}
		// 	return e, func() tea.Msg { return entrySwap{SelectedIndex: e.Index, TargetIndex: e.Index - 1} }
		// case 'j', tea.KeyDown:
		// 	if !msg.Key().Mod.Contains(tea.ModCtrl) {
		// 		break
		// 	}
		// 	return e, func() tea.Msg { return entrySwap{SelectedIndex: e.Index, TargetIndex: e.Index + 1} }
		case 'c':
			if msg.Key().Mod.Contains(tea.ModCtrl) {
				return e, tea.SetClipboard("test123")
			}
		case tea.KeyEnter:
			return e, func() tea.Msg { return entrySelected(e.Index) }
		}
	}
	e.Model, cmd = e.Model.Update(msg)
	return e, cmd
}
