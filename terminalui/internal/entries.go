package internal

import (
	"bytes"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/terminalui/card"
	"github.com/dkotik/cuebook/terminalui/list"
)

type entrySelected int

func selectEntryIndex(cmd tea.Cmd) tea.Cmd {
	if cmd == nil {
		return nil
	}
	switch msg := cmd().(type) {
	case list.SelectionMadeEvent:
		if msg.Index == 0 {
			return nil // TODO: launch front matter form
		} else if msg.Index < 0 {
			return nil
		}
		return func() tea.Msg { return entrySelected(msg.Index - 1) }
	default:
		return func() tea.Msg { return msg }
	}
}

type EntryList struct {
	tea.Model

	book     Book
	selected int // *patch.ByteAnchor
}

func (l EntryList) Init() (_ tea.Model, cmd tea.Cmd) {
	l.Model = list.New(entryListName)
	l.selected = -1
	return l, nil
}

type entryListCards []tea.Model

func (l EntryList) Update(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	switch msg := msg.(type) {
	case entrySelected:
		l.selected = int(msg)
		return l, nil
	case Book:
		l.book = msg
		return l, LoadEntries(msg.Document, l.selected, nil) // TODO: track patch changes
	case entryListCards:
		l.Model, cmd = l.Model.Init()
		l.Model, _ = l.Model.Update(list.Reset(entryListName)()) // TODO: event.ChainUpdate or list.SetItems{Index, Items}
		l.Model, _ = l.Model.Update(list.AddItems(entryListName, msg...)())
		l.Model, _ = l.Model.Update(list.ApplySelection(entryListName, l.selected)())
		return l, tea.Batch(cmd, tea.RequestWindowSize())
	case tea.KeyMsg:
		l.Model, cmd = l.Model.Update(msg)
		if msg.Key().Code == tea.KeyEnter {
			return l, selectEntryIndex(cmd)
		}
		return l, cmd
	default:
		l.Model, cmd = l.Model.Update(msg)
		return l, cmd
	}
}

const (
	entryListName = "cuebookEntryList"
)

func LoadEntries(book cuebook.Document, currentSelection int, r *cuebook.SourcePatchResult) tea.Cmd {
	return func() tea.Msg {
		total, err := book.Len()
		if err != nil {
			return err
		}
		cards := make([]tea.Model, 0, total+1)
		title := list.Title{
			Text:  book.Metadata().Title(),
			Style: lipgloss.NewStyle().Bold(true).Align(lipgloss.Left).Foreground(lipgloss.BrightRed),
		}
		cards = append(cards, title)

		index := 0
		selectIndex := -1
		for entry, err := range book.EachEntry() {
			if err != nil {
				return err
			}
			if r != nil && r.PrecedingDuplicates >= 0 {
				at := cuebook.GetByteSpanInSource(entry.Value)
				if !at.IsValid() {
					continue // TODO: handle
				}
				if bytes.Equal(r.ReplaceWith, r.Source[at.BeginsAt:at.EndsAt]) {
					selectIndex = index
					r.PrecedingDuplicates--
				}
				index++
			}
			cards = append(cards, card.New(entry.GetTitle(), entry.GetDescription()...))
		}
		if selectIndex >= 0 { // found matching bytes
			currentSelection = selectIndex // TODO: write a test for it
		}
		return entryListCards(cards)
		// return tea.BatchMsg{
		// 	tea.Sequence(
		// 		list.Reset(entryListName),
		// 		list.AddItems(entryListName, cards...),
		// 		tea.RequestWindowSize(),
		// 		list.ApplySelection(entryListName, currentSelection+1), // +1 for title
		// 	),
		// }
	}
}
