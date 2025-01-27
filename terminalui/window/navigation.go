package window

import (
	tea "github.com/charmbracelet/bubbletea/v2"
)

func (w window) back() (tea.Model, tea.Cmd) {
	if l := len(w.stack); l > 0 {
		l -= 1
		w.current = w.stack[l]
		w.stack = w.stack[:l]
		return w, nil
	}
	return w, tea.Quit
}
