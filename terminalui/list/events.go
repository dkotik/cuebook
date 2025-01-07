package list

import (
	"slices"

	tea "github.com/charmbracelet/bubbletea/v2"
)

type addItemsEvent struct {
	ListName string
	Items    []tea.Model
}

func AddItems(listName string, items ...tea.Model) tea.Cmd {
	return func() tea.Msg {
		items = slices.DeleteFunc(items, func(m tea.Model) bool {
			return m == nil
		})
		return addItemsEvent{
			ListName: listName,
			Items:    items,
		}
	}
}

type resetEvent struct {
	ListName string
}

func Reset(listName string) tea.Cmd {
	return func() tea.Msg {
		return resetEvent{ListName: listName}
	}
}

type SelectionActivateEvent bool

type filterEvent struct {
	SearchQuery string
}

type applySelectionEvent struct {
	ListName string
	Index    int
}

func ApplySelection(listName string, index int) tea.Cmd {
	return func() tea.Msg {
		return applySelectionEvent{
			ListName: listName,
			Index:    index,
		}
	}
}

func (l List) applySelection(index int) (tea.Model, tea.Cmd) {
	if len(l.Items) == 0 {
		return l, nil
	}
	var cmdPrevious, cmdNext tea.Cmd
	l.Items[l.SelectedIndex], cmdPrevious = l.Items[l.SelectedIndex].Update(SelectionActivateEvent(false))
	l.Items[index], cmdNext = l.Items[index].Update(SelectionActivateEvent(true))
	l.SelectedIndex = index
	// panic(index)
	return l, tea.Batch(cmdPrevious, cmdNext)
}

func ApplyFilter(searchQuery string) tea.Cmd {
	return func() tea.Msg {
		return filterEvent{SearchQuery: searchQuery}
	}
}
