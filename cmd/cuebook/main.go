package main

import (
	"context"
	"os"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/dkotik/cuebook/terminalui/file"
	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:  "cuebook",
		Usage: "edit lists of structured data items",
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

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		panic(err)
	}
}
