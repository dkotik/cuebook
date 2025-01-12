package window

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/dkotik/cuebook/terminalui/event"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

type (
	BackEvent    struct{}
	IsBusyEvent  bool
	SetBusyEvent bool

	localizerRequestEvent      struct{}
	commandContextRequestEvent struct{}
)

func WithBusySignal(cmd tea.Cmd) tea.Cmd {
	return tea.Sequence(
		func() tea.Msg { return SetBusyEvent(true) },
		cmd,
		func() tea.Msg { return SetBusyEvent(false) },
	)
}

func RequestCommandContext() tea.Cmd {
	return func() tea.Msg { return commandContextRequestEvent{} }
}

func RequestLocalizer() tea.Cmd {
	return func() tea.Msg { return localizerRequestEvent{} }
}

func SetLocalizer(lc *i18n.Localizer) tea.Cmd {
	return func() tea.Msg { return lc }
}

func (w window) Propagate(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	w.current, cmd = w.current.Update(msg)
	// if len(w.stack) > 0 {
	// 	cmd = tea.Batch(cmd, event.Propagate(msg, w.stack))
	// }
	return w, tea.Batch(cmd, event.Propagate(msg, w.stack), event.Propagate(msg, w.watchers))
}
