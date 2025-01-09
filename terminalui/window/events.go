package window

import (
	tea "github.com/charmbracelet/bubbletea/v2"
)

type (
	BackEvent    struct{}
	IsBusyEvent  bool
	SetBusyEvent bool
)

func WithBusySignal(cmd tea.Cmd) tea.Cmd {
	return tea.Sequence(
		func() tea.Msg { return SetBusyEvent(true) },
		cmd,
		func() tea.Msg { return SetBusyEvent(false) },
	)
}
