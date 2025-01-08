package internal

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/terminalui/file"
	"github.com/dkotik/cuebook/terminalui/list"
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
		return s, parseBook(msg)
	case parsedBook:
		s.Book = msg.Book
		s.Source = msg.Source
		s.EntryCount = 0
		s.FieldCount = 0
		return s, tea.Sequence(
			list.Count(entryListName),
			list.Count(entryFieldListName),
			func() tea.Msg { return updateLists{} },
		)
	case updateLists:
		// TODO: not needed, since selection and text update
		// events differentiate
		// creation vs update?
		// panic("issue new list or update if EntryCount is 0")
		// panic("issue new field list or update if FieldCount is 0")
		return s, nil
	case list.CountEvent:
		switch msg.ListName {
		case entryListName:
			s.EntryCount = msg.Count
		case entryFieldListName:
			s.FieldCount = msg.Count
		}
	case list.SelectionMadeEvent:
		switch msg.ListName {
		case entryListName:
		case entryFieldListName:
		}
	}
	s.Model, cmd = s.Model.Update(msg)
	return s, cmd
}
