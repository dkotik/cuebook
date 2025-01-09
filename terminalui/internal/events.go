package internal

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/terminalui/file"
	"github.com/dkotik/cuebook/terminalui/list"
	"github.com/dkotik/cuebook/terminalui/window"
)

type (
	parsedBook struct {
		Book   cuebook.CueBook
		Source []byte
	}

	updateLists struct{}
)

func (s state) Update(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	switch msg := msg.(type) {
	case file.ContentEvent:
		if s.IsEntryListAvailable() {
			return s, parseBook(msg)
		}
		return s, tea.Sequence(
			func() tea.Msg {
				return window.SwitchTo(list.New(entryListName))
			},
			parseBook(msg),
		)
	// case cuebook.SourcePatchResult:
	// 	s.Book = msg.Book
	// 	s.Source = msg.Source
	// 	return s, nil
	case parsedBook:
		s.Book = msg.Book
		s.Source = msg.Source
		return s, tea.Batch(
			LoadEntries(s.Book, s.SelectedEntryIndex-1),
			LoadFields(s.Book, s.SelectedEntryIndex-1),
		)
	case window.BackEvent:
		s.SelectedEntryIndex = -2
		s.SelectedFieldIndex = -2
		s.Model, cmd = s.Model.Update(msg)
		return s, tea.Batch(
			list.SelectedIndex(entryListName),
			list.SelectedIndex(entryFieldListName),
			cmd,
		)
	case list.SelectionMadeEvent:
		switch msg.ListName {
		case entryListName:
			s.SelectedEntryIndex = msg.Index
			s.Model, cmd = s.Model.Update(msg)
			return s, tea.Sequence(
				cmd,
				func() tea.Msg {
					return window.SwitchTo(list.New(entryFieldListName))
				},
				LoadFields(s.Book, s.SelectedEntryIndex-1),
			)
		case entryFieldListName:
			s.SelectedFieldIndex = msg.Index
		}
	case list.SelectedIndexEvent:
		switch msg.ListName {
		case entryListName:
			s.SelectedEntryIndex = msg.Index
		case entryFieldListName:
			s.SelectedFieldIndex = msg.Index
		}
	}
	s.Model, cmd = s.Model.Update(msg)
	return s, cmd
}
