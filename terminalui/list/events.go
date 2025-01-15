package list

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea/v2"
)

type addItemsEvent struct {
	Items []tea.Model
}

func AddItems(items ...tea.Model) tea.Cmd {
	return func() tea.Msg {
		for i, item := range items {
			if item == nil {
				return fmt.Errorf("list item #%d is nil", i)
			}
		}
		return addItemsEvent{
			Items: items,
		}
	}
}

type setItemsEvent []tea.Model

func SetItems(items ...tea.Model) tea.Cmd {
	return func() tea.Msg {
		for i, item := range items {
			if item == nil {
				return fmt.Errorf("list item #%d is nil", i)
			}
		}
		return setItemsEvent(items)
	}
}

type resetEvent struct {
	ListName string
}

func Reset() tea.Cmd {
	return func() tea.Msg {
		return resetEvent{}
	}
}

type SelectionHighlightEvent bool

type SelectionMadeEvent struct {
	Index int
}

type filterEvent struct {
	SearchQuery string
}

type applySelectionEvent struct {
	Index int
}

type selectedIndexRequestEvent struct {
}

type SelectedIndexEvent struct {
	Index int
}

type SwapOrderEvent struct {
	CurrentIndex int
	DesiredIndex int
}

func SelectedIndex(listName string) tea.Cmd {
	return func() tea.Msg {
		return selectedIndexRequestEvent{}
	}
}

func ApplySelection(index int) tea.Cmd {
	return func() tea.Msg {
		return applySelectionEvent{
			Index: index,
		}
	}
}

func (l List) applySelection(index int) (tea.Model, tea.Cmd) {
	total := len(l.Items)
	if total == 0 {
		return l, nil
	}
	// if index+1 > total {
	// 	index = total - 1
	// }

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
}

type CountEvent struct {
	Count int
}

func Count(listName string) tea.Cmd {
	return func() tea.Msg {
		return countRequestEvent{}
	}
}
