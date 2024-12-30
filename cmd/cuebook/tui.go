package main

import (
	"context"
	"os"

	"log/slog"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/terminalui"
	"github.com/dkotik/cuebook/terminalui/card"
	"github.com/dkotik/cuebook/terminalui/file"
	"github.com/dkotik/cuebook/terminalui/list"
)

func NewTerminalUI(ctx context.Context, filePath string) tea.Model {
	logFile, err := os.OpenFile("test/testdata/debug.log", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		panic(err)
	}
	go func() {
		<-ctx.Done()
		if err := logFile.Close(); err != nil {
			panic(err)
		}
	}()
	logger := slog.NewTextHandler(logFile, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	window := terminalui.New(
		file.New(filePath),
		// textarea.Textarea{
		// 	Label:    "Enter field",
		// 	Required: true,
		// },
		// list.New(card1, card2),
		// terminalui.NewSwitch(card1, card2, false),
		// card1,
		slog.New(logger).With("component", "bubbletea"),
	)

	return terminalui.NewDomainAdaptor(ctx,
		func(ctx context.Context, content file.ContentEvent) (tea.Msg, error) {
			book, err := cuebook.New(content)
			cards := make([]tea.Model, 0, book.Len()+1)
			// cards := make([]tea.Model, 0)
			if err != nil {
				return nil, err
			}
			cards = append(cards, list.Title{
				Text:  "skdjflkadksafkl ajs;jsadfadsjhsadkf sadjfas nsbdfjasdkf skjfhasjsdjkfbaisdasdhf ajsdlkjalkfadsl haskfhsafn saf 98-06458-6983-684356 3459843986-436-09358 035-0436093456-45869348-643568345683546834968-43560-4604564  9450684 90684036-03456",
				Style: lipgloss.NewStyle().Bold(true).Align(lipgloss.Left).Foreground(lipgloss.BrightRed),
			})
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
					} else {
						// description = append(description, field.String())
					}

					// TODO: slog invalid byte ranges?
				}

				cards = append(cards,
					terminalui.NewCached(
						card.New(entry.Title, description...),
					),
				)
			}

			return terminalui.SwitchTo(list.New(cards...)), nil
		},
		window)

}
