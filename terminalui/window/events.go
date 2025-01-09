package window

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

type (
	BackEvent    struct{}
	IsBusyEvent  bool
	SetBusyEvent bool

	localizerRequestEvent struct{}
)

func WithBusySignal(cmd tea.Cmd) tea.Cmd {
	return tea.Sequence(
		func() tea.Msg { return SetBusyEvent(true) },
		cmd,
		func() tea.Msg { return SetBusyEvent(false) },
	)
}

func RequestLocalizer() tea.Cmd {
	return func() tea.Msg { return localizerRequestEvent{} }
}

func SetLocalizer(lc *i18n.Localizer) tea.Cmd {
	return func() tea.Msg { return lc }
}
