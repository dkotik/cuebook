package internal

import (
	"context"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/terminalui/list"
	"github.com/dkotik/cuebook/terminalui/window"
)

func NewSelectionAdapter[T ~int](cmd tea.Cmd) tea.Cmd {
	if cmd == nil {
		return nil
	}
	switch msg := cmd().(type) {
	case list.SelectionMadeEvent:
		if msg.Index == 0 {
			return nil // TODO: launch front matter form
		} else if msg.Index < 0 {
			return nil
		}
		return func() tea.Msg { return T(msg.Index - 1) }
	default:
		return func() tea.Msg { return msg }
	}
}

func WithStateEventTransformers() window.Option {
	return window.WithWatchers(
		patchHistoryTracker{},
		// bookTracker{},
		state{},
	)
}

type state struct {
	Context            context.Context
	SelectedEntryIndex int
	SelectedFieldIndex int
	Document           cuebook.Document
	Source             []byte
	LastSourcePatch    *cuebook.SourcePatchResult
}

func (s state) IsReady() bool {
	return len(s.Source) > 0
}

func (s state) IsEntryListAvailable() bool {
	return s.SelectedEntryIndex != -2
}

func (s state) IsFieldListAvailable() bool {
	return s.SelectedFieldIndex != -2
}

func (s state) Init() (tea.Model, tea.Cmd) {
	s.SelectedEntryIndex = -2
	s.SelectedFieldIndex = -2
	return s, nil
}

func (s state) View() string { return "" }
