package internal

import (
	"bytes"
	"strings"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/metadata"
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

func displayMetadata(source []byte, md *metadata.Frontmatter) tea.Cmd {
	return func() tea.Msg {
		view := list.New(
			markdown.New(string(md.Source)),
			list.NewButton(&i18n.LocalizeConfig{
				DefaultMessage: &i18n.Message{
					ID:    "bookMetadataUpdateDescription",
					Other: "Update description",
				},
			}, updateMetadata(source, md)),
		)
		return window.SwitchTo(view)
	}
}

func updateMetadata(source []byte, md *metadata.Frontmatter) tea.Cmd {
	return func() tea.Msg {
		formWrapper, patchWrapper := NewPatchCloser("metadataPatch")
		form, err := textarea.New(
			textarea.WithLabel("Description"),
			textarea.WithValue(string(md.Source)),
			textarea.WithOnSubmitCommand(func(value string) tea.Cmd {
				return func() tea.Msg {
					b := &bytes.Buffer{}
					for _, line := range strings.Split(value, "\n") {
						_, _ = b.Write([]byte(`// `))
						_, _ = b.WriteString(line)
						_, _ = b.WriteRune('\n')
					}
					_, _ = b.WriteRune('\n') // skip a line after comments
					p, err := patch.UpdateRange(source, md.ByteRange, b.Bytes())
					if err != nil {
						return err
					}
					return patchWrapper(p)
				}
			}),
		)
		if err != nil {
			return err
		}
		return window.SwitchTo(formWrapper(form))
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
