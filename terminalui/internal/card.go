package internal

import (
	"cuelang.org/go/cue"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/terminalui/field"
	"github.com/dkotik/cuebook/terminalui/list"
)

const entryFieldListName = "cuebookEntryFieldList"

func LoadFields(book cuebook.CueBook, index int) tea.Cmd {
	if index < 0 {
		return nil
	}
	return func() tea.Msg {
		entry, err := cuebook.NewEntry(book.Value.LookupPath(cue.MakePath(cue.Index(index))))
		if err != nil {
			panic(err) // TODO: handle
		}
		fields := make([]tea.Model, 0, len(entry.Fields)+len(entry.Details)+1)
		fields = append(fields, list.Title{
			Text:  book.Metadata().Title(),
			Style: lipgloss.NewStyle().Bold(true).Align(lipgloss.Left).Foreground(lipgloss.BrightRed),
		})
		for _, f := range entry.Fields {
			fields = append(fields, field.New(f.Name, f.String()))
		}
		for _, f := range entry.Details {
			fields = append(fields, field.New(f.Name, f.String()))
		}

		return tea.BatchMsg{
			tea.Sequence(
				list.Reset(entryFieldListName),
				list.AddItems(entryFieldListName, fields...),
			),
		}
	}
}
