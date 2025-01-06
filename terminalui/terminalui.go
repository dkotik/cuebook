/*
Package terminalui provides user interface built on Charm.
*/
package terminalui

import (
	"cmp"
	"log/slog"
	"time"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/davecgh/go-spew/spew"
)

func New(initial tea.Model, logger *slog.Logger) tea.Model {
	if initial == nil {
		panic("initial model is nil")
	}
	return window{
		current: initial,
		logger:  cmp.Or(logger, slog.Default()),
	}
}

type window struct {
	current tea.Model
	stack   []tea.Model
	size    tea.WindowSizeMsg
	busy    uint8
	logger  *slog.Logger
}

func (w window) Init() (_ tea.Model, cmd tea.Cmd) {
	w.stack = make([]tea.Model, 0, 5)
	w.current, cmd = w.current.Init()
	return w, cmd
}

func (w window) back() (tea.Model, tea.Cmd) {
	if l := len(w.stack); l > 0 {
		l -= 1
		w.current = w.stack[l]
		w.stack = w.stack[:l]
		var cmd tea.Cmd
		w.current, cmd = w.current.Update(w.size)
		return w, cmd
	}
	return w, tea.Quit
}

func (w window) Update(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	go w.logger.Debug(spew.Sdump(msg))
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		w.size = msg
	case SwitchTo:
		w.stack = append(w.stack, w.current)
		var cmdInit tea.Cmd
		w.current, cmdInit = msg.Init()
		w.current, cmd = w.current.Update(w.size)
		return w, tea.Batch(cmd, cmdInit)
	case BackEvent:
		return w.back()
	case setBusyEvent:
		if bool(msg) {
			w.busy++
			return w, func() tea.Msg {
				return IsBusyEvent(true)
			}
		} else {
			w.busy--
			if w.busy == 0 {
				return w, func() tea.Msg {
					// delay busy reset
					// to allow the animation to play longer
					time.Sleep(time.Second)
					return IsBusyEvent(false)
				}
			}
		}
		return w, nil
	case tea.KeyMsg:
		if w.busy != 0 {
			return w, nil // drop event if busy
		}
		switch msg.Key().Code {
		case tea.KeyEscape:
			return w.back()
		}
		// fallthrough
		w.current, cmd = w.current.Update(msg)
		return w, cmd
	case tea.MouseMsg, tea.CursorPositionMsg, tea.PasteStartMsg, tea.PasteEndMsg, tea.PasteMsg: // input events go to current model only
		if w.busy != 0 {
			return w, nil // drop event if busy
		}
		w.current, cmd = w.current.Update(msg)
		return w, cmd
	}
	w.current, cmd = w.current.Update(msg)
	if len(w.stack) > 0 {
		cmd = tea.Batch(cmd, Propagate(msg, w.stack))
	}
	return w, cmd
}

func (w window) View() string {
	return w.current.View()
}
