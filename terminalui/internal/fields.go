package internal

import (
	"fmt"

	"cuelang.org/go/cue"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/terminalui/field"
	"github.com/dkotik/cuebook/terminalui/file"
	"github.com/dkotik/cuebook/terminalui/list"
	"github.com/dkotik/cuebook/terminalui/textarea"
	"github.com/dkotik/cuebook/terminalui/window"
)

const entryFieldListName = "cuebookEntryFieldList"

func IssueFieldPatch(book cuebook.CueBook, source []byte, entryIndex, fieldIndex int, value string) tea.Cmd {
	return func() tea.Msg {
		f, err := book.GetField(entryIndex, fieldIndex)
		if err != nil {
			return err
		}
		patch, err := f.WithStringValue(source, value)
		if err != nil {
			return err
		}
		result, err := patch.Apply(source)
		if err != nil {
			return err
		}
		return file.SaveEvent(result.Source)
	}
}

func SwitchToFieldForm(book cuebook.CueBook, entryIndex, fieldIndex int) tea.Cmd {
	return func() tea.Msg {
		f, err := book.GetField(entryIndex, fieldIndex)
		if err != nil {
			return err
		}
		return window.SwitchTo(textarea.New(f.Name, f.String(), true))
	}
}

func LoadFields(book cuebook.CueBook, index int) tea.Cmd {
	if index < 0 {
		return nil
	}
	return func() tea.Msg {
		entry, err := cuebook.NewEntry(book.Value.LookupPath(cue.MakePath(cue.Index(index))))
		if err != nil {
			return err
		}
		total, err := book.Len()
		if err != nil {
			return err
		}
		fields := make([]tea.Model, 0, len(entry.Fields)+len(entry.Details)+1)
		fields = append(fields, list.Title{
			Text:  entry.GetTitle() + fmt.Sprintf(" › %d/%d", index+1, total),
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
				list.ApplySelection(entryFieldListName, index+1), // +1 for title
			),
		}
	}
}
