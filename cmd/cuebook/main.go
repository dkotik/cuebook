package main

import (
	"context"
	"os"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:  "cuebook",
		Usage: "edit lists of structured data items",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			ui, err := NewTerminalUI(ctx, "test/testdata/simple.cue")
			if err != nil {
				return err
			}

			_, err = tea.NewProgram(
				ui,
				tea.WithContext(ctx),
				tea.WithAltScreen(),
				tea.WithMouseCellMotion(),
			).Run()
			return err
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		panic(err)
	}
}
