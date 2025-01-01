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
			if err != nil {
				return nil, err
			}
			cards := make([]tea.Model, 0, book.Len()+1)
			cards = append(cards, list.Title{
				Text:  book.Metadata().Title(),
				Style: lipgloss.NewStyle().Bold(true).Align(lipgloss.Left).Foreground(lipgloss.BrightRed),
			})
			for entry, err := range book.EachEntry() {
				if err != nil {
					return nil, err
				}
				// index := len(cards)
				cards = append(cards,
					terminalui.NewDomainAdaptor(ctx,
						func(ctx context.Context, keyPress tea.KeyMsg) (tea.Msg, error) {
							if keyPress.Key().Code == tea.KeyEnter {
								return terminalui.SwitchTo(
									card.New(">>" + entry.GetTitle()),
								), nil
							}
							return nil, nil
						},
						card.New(entry.GetTitle(), entry.GetDescription()...)),
				)
			}
			return terminalui.SwitchTo(list.New(cards...)), nil
		},
		window)
}
