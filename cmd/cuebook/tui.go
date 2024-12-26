package main

import (
	"context"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/terminalui"
	"github.com/dkotik/cuebook/terminalui/card"
	"github.com/dkotik/cuebook/terminalui/file"
	"github.com/dkotik/cuebook/terminalui/list"
)

func NewTerminalUI(ctx context.Context, filePath string) tea.Model {
	window := terminalui.New(
		file.New(filePath),
		// textarea.Textarea{
		// 	Label:    "Enter field",
		// 	Required: true,
		// },
		// list.New(card1, card2),
		// terminalui.NewSwitch(card1, card2, false),
		// card1,
	)

	return terminalui.NewDomainAdaptor(ctx,
		func(ctx context.Context, content file.ContentEvent) (tea.Msg, error) {
			book, err := cuebook.New(content)
			cards := make([]tea.Model, 0, book.Len())
			if err != nil {
				return nil, err
			}
			for entry, err := range book.EachEntry() {
				if err != nil {
					return nil, err
				}
				var description []string
				var byteRange cuebook.SourceByteRange
				for _, field := range entry.Fields {
					byteRange = cuebook.GetByteSpanInSource(field.Value)
					if byteRange.IsValid() {
						description = append(description, field.String())
					}
					// TODO: slog invalid byte ranges?
				}

				cards = append(cards, card.New(entry.Title, description...))
			}

			return terminalui.SwitchTo(list.New(cards...)), nil
		},
		window)

}
