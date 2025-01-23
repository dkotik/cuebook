package internal

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/dkotik/cuebook/terminalui/list"
	"github.com/dkotik/cuebook/terminalui/window"
)

// TODO: deprecate in favor of embedded function calls on models
func NewListItemHighlightAdaptor[T ~int](cmd tea.Cmd) tea.Cmd {
	if cmd == nil {
		return nil
	}
	switch msg := cmd().(type) {
	case list.HighlightEvent:
		return func() tea.Msg { return T(msg) }
	default:
		return func() tea.Msg { return msg }
	}
}

func WithStateEventTransformers() window.Option {
	return window.WithWatchers(
		patchHistoryTracker{},
		flashAnnouncer{},
	)
}
