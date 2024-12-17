/*
Package terminalui provides user interface built on Charm.
*/
package terminalui

import (
	tea "github.com/charmbracelet/bubbletea/v2"
)

type Back struct{}

type window struct {
	model tea.Model
	stack []tea.Model
}

func New(initial tea.Model) tea.Model {
	return window{model: initial}
}

func (w window) Init() (_ tea.Model, cmd tea.Cmd) {
	w.model, cmd = w.model.Init()
	return w, cmd
}

func (w window) back() (tea.Model, tea.Cmd) {
	if l := len(w.stack); l > 0 {
		l--
		w.model = w.stack[l]
		w.stack = w.stack[:l]
		return w, nil
	}
	return w, tea.Quit
}

func (w window) Update(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	switch msg := msg.(type) {
	case SwitchTo:
		w.stack = append(w.stack, msg)
		w.model = msg
		return w, nil
	case Back:
		return w.back()
	case tea.KeyMsg:
		switch msg.Key().Code {
		case tea.KeyEscape:
			return w.back()
		}
	}
	w.model, cmd = w.model.Update(msg)
	return w, cmd
}

func (w window) View() string {
	return w.model.View()
}
