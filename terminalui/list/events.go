package list

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea/v2"
)

type addItemsEvent struct {
	ListName string
	Items    []tea.Model
}

func AddItems(listName string, items ...tea.Model) tea.Cmd {
	return func() tea.Msg {
		for i, item := range items {
			if item == nil {
				return fmt.Errorf("list item #%d is nil", i)
			}
		}
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

type SelectionHighlightEvent bool

type SelectionMadeEvent struct {
	ListName string
	Index    int
}

type filterEvent struct {
	SearchQuery string
}

type applySelectionEvent struct {
	ListName string
	Index    int
}

type selectedIndexRequestEvent struct {
	ListName string
}

type SelectedIndexEvent struct {
	ListName string
	Index    int
}

type SwapOrderEvent struct {
	ListName     string
	CurrentIndex int
	DesiredIndex int
}

func SelectedIndex(listName string) tea.Cmd {
	return func() tea.Msg {
		return selectedIndexRequestEvent{ListName: listName}
	}
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
	l.Items[l.SelectedIndex], cmdPrevious = l.Items[l.SelectedIndex].Update(SelectionHighlightEvent(false))
	l.Items[index], cmdNext = l.Items[index].Update(SelectionHighlightEvent(true))
	l.SelectedIndex = index
	return l, tea.Batch(cmdPrevious, cmdNext)
}

func ApplyFilter(searchQuery string) tea.Cmd {
	return func() tea.Msg {
		return filterEvent{SearchQuery: searchQuery}
	}
}

type countRequestEvent struct {
	ListName string
}

type CountEvent struct {
	ListName string
	Count    int
}

func Count(listName string) tea.Cmd {
	return func() tea.Msg {
		return countRequestEvent{
			ListName: listName,
		}
	}
}
