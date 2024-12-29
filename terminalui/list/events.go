package list

import tea "github.com/charmbracelet/bubbletea/v2"

type AddEvent struct{}

type SelectionActivateEvent bool

type filterEvent struct {
	SearchQuery string
}

type applySelectionEvent struct {
	Index int
}

func ApplySelection(index int) tea.Cmd {
	return func() tea.Msg {
		return applySelectionEvent{Index: index}
	}
}

func (l List) applySelection(index int) (tea.Model, tea.Cmd) {
	var cmdPrevious, cmdNext tea.Cmd
	l.Items[l.SelectedIndex], cmdPrevious = l.Items[l.SelectedIndex].Update(SelectionActivateEvent(false))
	l.Items[index], cmdNext = l.Items[index].Update(SelectionActivateEvent(true))
	l.SelectedIndex = index
	return l, tea.Batch(cmdPrevious, cmdNext)
}

func ApplyFilter(searchQuery string) tea.Cmd {
	return func() tea.Msg {
		return filterEvent{SearchQuery: searchQuery}
	}
}
