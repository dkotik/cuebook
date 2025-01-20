package internal

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/patch"
	"github.com/dkotik/cuebook/terminalui/file"
	"github.com/dkotik/cuebook/terminalui/window"
)

func parseBook(source []byte) tea.Cmd {
	return func() tea.Msg {
		document, err := cuebook.New(source)
		if err != nil {
			return err
		}
		return patch.Result{
			Document: document,
			Source:   source,
			Hash:     1, // TODO: think if this is smart; needed so that first value override nil
		}
	}
}

func ParseFileToBookAndCreateEntryListIfNeeded(cmd tea.Cmd) tea.Cmd {
	switch msg := cmd().(type) {
	case file.UpdateEvent:
		return parseBook(msg)
	case file.ContentEvent:
		return tea.Sequence(
			func() tea.Msg { return window.SwitchTo(EntryList{}) },
			parseBook(msg),
		)
	default:
		return func() tea.Msg { return msg }
	}
}
