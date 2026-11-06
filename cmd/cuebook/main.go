package main

import (
	"context"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/dkotik/cuebook/terminalui"
	"github.com/urfave/cli/v3"
)

func main() {
	if err := newCLICommand().Run(context.Background(), os.Args); err != nil {
		panic(err)
	}
}

func newCLICommand() *cli.Command {
	return &cli.Command{
		Name:      "cuebook",
		Usage:     "edit lists of structured data items",
		ArgsUsage: "<file>",
		Commands:  []*cli.Command{newServeCommand()},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() == 0 {
				return cli.ShowAppHelp(cmd)
			}
			if cmd.NArg() != 1 {
				return fmt.Errorf("cuebook requires exactly one file argument, got %d", cmd.NArg())
			}
			filePath := cmd.Args().First()

			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			ui, err := NewTerminalUI(ctx)
			if err != nil {
				return err
			}

			p := tea.NewProgram(
				ui,
				tea.WithContext(ctx),
			)
			go func() {
				p.Send(terminalui.LoadFileEvent(filePath))
			}()
			_, err = p.Run()

			return err
		},
	}
}
