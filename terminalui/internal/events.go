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
		Book   cuebook.Document
		Source []byte
	}
)

func (s state) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case file.ContentEvent:
		// panic("w")
		if s.IsEntryListAvailable() {
			return s, parseBook(msg)
		}
		return s, tea.Sequence(
			func() tea.Msg {
				return window.SwitchTo(list.New(entryListName))
			},
			list.SelectedIndex(entryListName),
			parseBook(msg),
		)
	case cuebook.SourcePatchResult:
		s.LastSourcePatch = &msg
		return s, file.Save(msg.Source)
	case parsedBook:
		s.Document = msg.Book
		s.Source = msg.Source
		lastSourcePatch := s.LastSourcePatch
		s.LastSourcePatch = nil
		return s, tea.Batch(
			event.If(s.IsEntryListAvailable(),
				LoadEntries(s.Document, s.SelectedEntryIndex-1, lastSourcePatch),
			),
			event.If(s.IsFieldListAvailable(),
				LoadFields(s.Document, s.SelectedEntryIndex-1),
				// func() tea.Msg {
				// 	return window.BackEvent{}
				// },
			),
		)
	case textarea.OnChangeEvent:
		if msg.TextAreaName == fieldEditingTextAreaName {
			return s, IssueFieldPatch(s.Document, s.Source, s.SelectedEntryIndex-1, s.SelectedFieldIndex-1, msg.Value)
		}
		if msg.TextAreaName == fieldAddingTextAreaName {
			// TODO: create patch using a list of field: value line pairs
		}
	case window.BackEvent:
		s.SelectedEntryIndex = -2
		s.SelectedFieldIndex = -2
		return s, tea.Sequence(
			list.SelectedIndex(entryListName),
			list.SelectedIndex(entryFieldListName),
		)
	case list.SelectionMadeEvent:
		switch msg.ListName {
		case entryListName:
			if msg.Index == 0 {
				msg.Index = 1 // TODO: allow viewing and editing frontmatter
			}
			s.SelectedEntryIndex = msg.Index
			return s, tea.Sequence(
				func() tea.Msg {
					return window.SwitchTo(list.New(entryFieldListName))
				},
				LoadFields(s.Document, s.SelectedEntryIndex-1),
			)
		case entryFieldListName:
			if msg.Index == 0 { // selecting title jumps to field
				return s, list.ApplySelection(entryFieldListName, 1)
			}
			s.SelectedFieldIndex = msg.Index
			return s, SwitchToFieldForm(s.Document, s.SelectedEntryIndex-1, s.SelectedFieldIndex-1)
		}
	case list.SelectedIndexEvent:
		switch msg.ListName {
		case entryListName:
			s.SelectedEntryIndex = msg.Index
		case entryFieldListName:
			s.SelectedFieldIndex = msg.Index
		}
	case list.SwapOrderEvent:
		switch msg.ListName {
		case entryListName:
			msg.CurrentIndex--
			msg.DesiredIndex--
			return s, tea.Batch(
				func() tea.Msg {
					original, err := s.Document.GetValue(msg.CurrentIndex)
					if err != nil {
						return err
					}
					swapWith, err := s.Document.GetValue(msg.DesiredIndex)
					if err != nil {
						return err
					}
					result, err := cuebook.SwapBytes(s.Source, original, swapWith)
					if err != nil {
						return err
					}
					return result
				},
			)
		case entryFieldListName:
			msg.CurrentIndex--
			msg.DesiredIndex--
			s.SelectedFieldIndex = msg.CurrentIndex // to trigger s.IsFieldListAvailable()
			return s, tea.Batch(
				func() tea.Msg {
					value, err := s.Document.GetValue(s.SelectedEntryIndex - 1)
					if err != nil {
						return err
					}
					entry, err := cuebook.NewEntry(value)
					if err != nil {
						return err
					}
					original, err := entry.GetField(msg.CurrentIndex)
					if err != nil {
						return err
					}
					swapWith, err := entry.GetField(msg.DesiredIndex)
					if err != nil {
						return err
					}
					result, err := cuebook.SwapBytes(s.Source, original.Value, swapWith.Value)
					if err != nil {
						return err
					}
					return result
				},
				// LoadFields(s.Document, s.SelectedEntryIndex-1), // TODO: does not update view
				// LoadEntries(s.Document, s.SelectedEntryIndex-1, nil),
			)
		}
	case tea.KeyMsg:
		if msg.Key().Code == 'x' && msg.Key().Mod == tea.ModCtrl {
			return s, func() tea.Msg {
				value, err := s.Document.GetValue(s.SelectedEntryIndex - 1)
				if err != nil {
					panic("out of range?") // TODO: handle
				}
				entry, err := cuebook.NewEntry(value)
				if err != nil {
					return err
				}
				patch, err := entry.Delete(s.Source)
				if err != nil {
					return err
				}
				result, err := patch.Apply(s.Source)
				if err != nil {
					return err
				}
				panic(string(result.Source))
			}
		}
	case error:
		panic(msg) // TODO: handle with care
	}
	return s, nil
}
