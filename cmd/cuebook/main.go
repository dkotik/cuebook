package main

import (
	"context"
	"os"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/dkotik/cuebook/terminalui/file"
	"github.com/urfave/cli/v3"
)

func main() {
	if err := newCLICommand().Run(context.Background(), os.Args); err != nil {
		panic(err)
	}
}

func newCLICommand() *cli.Command {
	return &cli.Command{
		Name:     "cuebook",
		Usage:    "edit lists of structured data items",
		Commands: []*cli.Command{newServeCommand()},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			ui, err := NewTerminalUI(ctx)
			if err != nil {
				return err
			}

			p := tea.NewProgram(
				ui,
				tea.WithContext(ctx),
				tea.WithAltScreen(),
				tea.WithMouseCellMotion(),
			)
			go func() {
				p.Send(file.LoadEvent("test/testdata/simple.cue"))
			}()
			_, err = p.Run()

			return err
		},
	}
}
