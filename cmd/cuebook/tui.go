package main

import (
	"context"
	"os"

	"log/slog"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/dkotik/cuebook/terminalui"
	"github.com/dkotik/cuebook/terminalui/file"
	"github.com/dkotik/cuebook/terminalui/window"
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

	w, err := window.New(
		window.WithCommandContext(ctx),
		window.WithInitialModels(file.New(filePath)),
		window.WithLogger(slog.New(logger).With("component", "bubbletea")),
	)
	if err != nil {
		panic(err)
	}

	return terminalui.NewWithCueState(w)
}
