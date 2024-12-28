package list

import tea "github.com/charmbracelet/bubbletea/v2"

type AddEvent struct{}

type filterEvent struct {
	SearchQuery string
}

type selectEvent struct {
	Index int
}

func NewSelection(index int) tea.Cmd {
	return func() tea.Msg {
		return selectEvent{Index: index}
	}
}

func NewFilter(searchQuery string) tea.Cmd {
	return func() tea.Msg {
		return filterEvent{SearchQuery: searchQuery}
	}
}
