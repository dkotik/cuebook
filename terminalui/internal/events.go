package internal

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/dkotik/cuebook/terminalui/list"
)

func (s state) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case list.SwapOrderEvent:
		// switch msg.ListName {
		// case entryListName:
		// 	msg.CurrentIndex--
		// 	msg.DesiredIndex--
		// 	return s, tea.Batch(
		// 		func() tea.Msg {
		// 			original, err := s.Document.GetValue(msg.CurrentIndex)
		// 			if err != nil {
		// 				return err
		// 			}
		// 			swapWith, err := s.Document.GetValue(msg.DesiredIndex)
		// 			if err != nil {
		// 				return err
		// 			}
		// 			result, err := cuebook.SwapBytes(s.Source, original, swapWith)
		// 			if err != nil {
		// 				return err
		// 			}
		// 			return result
		// 		},
		// 	)
		// case entryFieldListName:
		// 	msg.CurrentIndex--
		// 	msg.DesiredIndex--
		// 	s.SelectedFieldIndex = msg.CurrentIndex // to trigger s.IsFieldListAvailable()
		// 	return s, tea.Batch(
		// 		func() tea.Msg {
		// 			value, err := s.Document.GetValue(s.SelectedEntryIndex - 1)
		// 			if err != nil {
		// 				return err
		// 			}
		// 			entry, err := cuebook.NewEntry(value)
		// 			if err != nil {
		// 				return err
		// 			}
		// 			original, err := entry.GetField(msg.CurrentIndex)
		// 			if err != nil {
		// 				return err
		// 			}
		// 			swapWith, err := entry.GetField(msg.DesiredIndex)
		// 			if err != nil {
		// 				return err
		// 			}
		// 			result, err := cuebook.SwapBytes(s.Source, original.Value, swapWith.Value)
		// 			if err != nil {
		// 				return err
		// 			}
		// 			return result
		// 		},
		// 		// LoadFields(s.Document, s.SelectedEntryIndex-1), // TODO: does not update view
		// 		// LoadEntries(s.Document, s.SelectedEntryIndex-1, nil),
		// 	)
		// }
	case error:
		panic(msg) // TODO: handle with care
	}
	return s, nil
}
