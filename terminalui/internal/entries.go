package internal

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/terminalui/card"
	"github.com/dkotik/cuebook/terminalui/list"
)

const (
	entryListName = "cuebookEntryList"
)

func LoadEntries(book cuebook.CueBook, currentSelection int) tea.Cmd {
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
		// selectIndex := -1
		for entry, err := range book.EachEntry() {
			if err != nil {
				return err
			}
			index++
			// if r.ReplaceWith != nil && r.PrecedingDuplicates >= 0 {
			// 	at := cuebook.GetByteSpanInSource(entry.Value)
			// 	if !at.IsValid() {
			// 		continue // TODO: handle
			// 	}
			// 	if bytes.Equal(r.ReplaceWith, r.Source[at.BeginsAt:at.EndsAt]) {
			// 		selectIndex = index
			// 		// panic(index)
			// 		r.PrecedingDuplicates--
			// 	}
			// }
			cards = append(cards, card.New(entry.GetTitle(), entry.GetDescription()...))
		}
		return tea.BatchMsg{
			tea.Sequence(
				list.Reset(entryListName),
				list.AddItems(entryListName, cards...),
				tea.RequestWindowSize(),
				list.ApplySelection(entryListName, currentSelection+1), // +1 for title
			),
		}
	}
}
