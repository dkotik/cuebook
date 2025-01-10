package internal

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/terminalui/event"
	"github.com/dkotik/cuebook/terminalui/file"
	"github.com/dkotik/cuebook/terminalui/textarea"
	"github.com/dkotik/cuebook/terminalui/window"
)

var closeOnSuccessfulFileSave = event.NewAdaptor(func(m tea.Model, source file.ContentEvent) (tea.Model, tea.Cmd) {
	// TODO: this could be more sophisticated, if source is matched first to s.LastPatch
	// to account for changes made by other processes
	return m, func() tea.Msg { return window.BackEvent{} }
})

func SwitchToFieldForm(book cuebook.Document, entryIndex, fieldIndex int) tea.Cmd {
	return func() tea.Msg {
		f, err := book.GetField(entryIndex, fieldIndex)
		if err != nil {
			return err
		}
		return window.SwitchTo(
			closeOnSuccessfulFileSave(
				textarea.New(fieldEditingTextAreaName, f.Name, f.String(), true),
			),
		)
	}
}
