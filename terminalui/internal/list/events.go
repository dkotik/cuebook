package list

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea/v2"
)

type (
	applySelectionEvent int
	HighlightHintEvent  bool
	// HighlightEvent      int // TODO: replicate

	setItemsEvent []tea.Model
	addItemsEvent []tea.Model
	resetEvent    struct{}
)

func AddItems(items ...tea.Model) tea.Cmd {
	return func() tea.Msg {
		for i, item := range items {
			if item == nil {
				return fmt.Errorf("list item #%d is nil", i)
			}
		}
		return addItemsEvent(items)
	}
}

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

func Reset() tea.Cmd {
	return func() tea.Msg {
		return resetEvent{}
	}
}

type filterEvent struct {
	SearchQuery string
}

type SwapOrderEvent struct {
	CurrentIndex int
	DesiredIndex int
}

func ApplySelection(index int) tea.Cmd {
	return func() tea.Msg {
		return applySelectionEvent(index)
	}
}

func (l List) applySelection(index int) (tea.Model, tea.Cmd) {
	total := len(l.Items)
	if total == 0 {
		return l, nil
	}

	var cmdPrevious, cmdNext tea.Cmd
	// if l.SelectedIndex >= 0 && l.SelectedIndex < total {
	l.Items[l.SelectedIndex], cmdPrevious = l.Items[l.SelectedIndex].Update(HighlightHintEvent(false))
	// }
	l.Items[index], cmdNext = l.Items[index].Update(HighlightHintEvent(true))
	l.SelectedIndex = index
	return l, tea.Batch(cmdPrevious, cmdNext) // , func() tea.Msg { return HighlightEvent(index) })
}

func ApplyFilter(searchQuery string) tea.Cmd {
	return func() tea.Msg {
		return filterEvent{SearchQuery: searchQuery}
	}
}

type countRequestEvent struct{}

type CountEvent struct {
	Count int
}

func Count(listName string) tea.Cmd {
	return func() tea.Msg {
		return countRequestEvent{}
	}
}
