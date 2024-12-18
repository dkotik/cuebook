/*
Package terminalui provides user interface built on Charm.
*/
package terminalui

import (
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/davecgh/go-spew/spew"
)

type (
	ErrorEvent   error
	BackEvent    struct{}
	IsBusyEvent  bool
	setBusyEvent bool
)

func WithBusySignal(cmd tea.Cmd) tea.Cmd {
	return tea.Sequence(
		func() tea.Msg { return setBusyEvent(true) },
		cmd,
		func() tea.Msg { return setBusyEvent(false) },
	)
}

type window struct {
	model tea.Model
	stack []tea.Model
	busy  uint8
}

func New(initial tea.Model) tea.Model {
	return window{model: initial}
}

func (w window) Init() (_ tea.Model, cmd tea.Cmd) {
	// TODO: logging should be behind a flag?
	var err error
	logFile, err = os.OpenFile("test/testdata/debug.log", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		os.Exit(1)
	}

	w.model, cmd = w.model.Init()
	return w, WithBusySignal(cmd)
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

var logFile *os.File

func (w window) Update(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	spew.Fdump(logFile, time.Now().Format("04:05.000"), msg)

	switch msg := msg.(type) {
	case SwitchTo:
		w.stack = append(w.stack, msg)
		w.model = msg
	case BackEvent:
		return w.back()
	case setBusyEvent:
		if bool(msg) {
			w.busy++
			w.model, cmd = w.model.Update(IsBusyEvent(true))
			return w, cmd
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
	case IsBusyEvent:
		w.model, cmd = w.model.Update(IsBusyEvent(w.busy != 0))
	case tea.KeyMsg:
		switch msg.Key().Code {
		case tea.KeyEscape:
			return w.back()
		default:
			if w.busy == 0 {
				// forward key events only if not busy
				w.model, cmd = w.model.Update(msg)
			}
		}
	default:
		w.model, cmd = w.model.Update(msg)
	}
	return w, cmd
}

func (w window) View() string {
	return w.model.View()
}
