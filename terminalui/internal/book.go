package internal

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/patch"
	"github.com/dkotik/cuebook/terminalui/file"
	"github.com/dkotik/cuebook/terminalui/textarea"
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
		}
	}
}

func updateMetadata(doc cuebook.Document) tea.Cmd {
	return func() tea.Msg {
		form, err := textarea.New(
			textarea.WithLabel("Description"),
			textarea.WithValue(doc.Metadata().Description()),
		)
		if err != nil {
			return err
		}
		return window.SwitchTo(form)
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
