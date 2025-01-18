package internal

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/patch"
	"github.com/dkotik/cuebook/terminalui/file"
	"github.com/dkotik/cuebook/terminalui/list"
	"github.com/dkotik/cuebook/terminalui/markdown"
	"github.com/dkotik/cuebook/terminalui/textarea"
	"github.com/dkotik/cuebook/terminalui/window"
	"github.com/nicksnyder/go-i18n/v2/i18n"
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

func displayMetadata(doc cuebook.Document) tea.Cmd {
	return func() tea.Msg {
		view := list.New(
			markdown.New(string(doc.Metadata().Source)),
			list.NewButton(&i18n.LocalizeConfig{
				DefaultMessage: &i18n.Message{
					ID:    "bookMetadataUpdateDescription",
					Other: "Update description",
				},
			}, updateMetadata(doc)),
		)
		return window.SwitchTo(view)
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
