package internal

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/terminalui/event"
	"github.com/dkotik/cuebook/terminalui/file"
	"github.com/dkotik/cuebook/terminalui/list"
	"github.com/dkotik/cuebook/terminalui/textarea"
	"github.com/dkotik/cuebook/terminalui/window"
)

type (
	parsedBook struct {
		Book   cuebook.CueBook
		Source []byte
	}

	// updateLists struct{}
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
	case cuebook.SourcePatchResult:
		s.LastSourcePatch = &msg
		return s, file.Save(msg.Source)
	case parsedBook:
		s.Book = msg.Book
		s.Source = msg.Source
		lastSourcePatch := s.LastSourcePatch
		s.LastSourcePatch = nil
		return s, tea.Batch(
			LoadEntries(s.Book, s.SelectedEntryIndex-1, lastSourcePatch),
			event.If(
				s.IsFieldListAvailable(),
				LoadFields(s.Book, s.SelectedEntryIndex-1),
				func() tea.Msg {
					return window.BackEvent{}
				},
			),
		)
	case textarea.OnChangeEvent:
		if s.IsFieldListAvailable() && msg.TextAreaName == fieldEditingTextAreaName {
			return s, IssueFieldPatch(s.Book, s.Source, s.SelectedEntryIndex-1, s.SelectedFieldIndex-1, msg.Value)
		}
		if msg.TextAreaName == fieldAddingTextAreaName {
			// TODO: create patch using a list of field: value line pairs
		}
	case window.BackEvent:
		s.SelectedEntryIndex = -2
		s.SelectedFieldIndex = -2
		s.Model, cmd = s.Model.Update(msg)
		return s, tea.Sequence(
			list.SelectedIndex(entryListName),
			list.SelectedIndex(entryFieldListName),
			cmd,
		)
	case list.SelectionMadeEvent:
		switch msg.ListName {
		case entryListName:
			if msg.Index == 0 {
				msg.Index = 1 // TODO: allow viewing and editing frontmatter
			}
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
			if msg.Index == 0 { // selecting title jumps to field
				return s, list.ApplySelection(entryFieldListName, 1)
			}
			s.SelectedFieldIndex = msg.Index
			s.Model, cmd = s.Model.Update(msg)
			return s, tea.Batch(cmd,
				SwitchToFieldForm(s.Book, s.SelectedEntryIndex-1, s.SelectedFieldIndex-1))
		}
	case list.SelectedIndexEvent:
		switch msg.ListName {
		case entryListName:
			s.SelectedEntryIndex = msg.Index
		case entryFieldListName:
			s.SelectedFieldIndex = msg.Index
		}
	case error:
		panic(msg) // TODO: handle with care
	}
	s.Model, cmd = s.Model.Update(msg)
	return s, cmd
}
