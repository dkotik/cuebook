package main

import (
	"context"
	"os"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/dkotik/cuebook/terminalui"
	"github.com/dkotik/cuebook/terminalui/textarea"
	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:  "cuebook",
		Usage: "edit lists of structured data items",
		Action: func(ctx context.Context, cmd *cli.Command) (err error) {
			// card1, err := card.New("Title", "sdf sdf sdf", "sdf sdfsdf sdf")
			// if err != nil {
			// 	return err
			// }

			// card2, err := card.New("Title2", "?????", "????")
			// if err != nil {
			// 	return err
			// }

			_, err = tea.NewProgram(
				terminalui.New(
					textarea.Textarea{
						Label:    "Enter field",
						Required: true,
					},
					// list.New(card1, card2),
					// terminalui.NewSwitch(card1, card2, false),
					// card1,
				),
				tea.WithContext(ctx),
				tea.WithAltScreen(),
			).Run()
			return err
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		panic(err)
	}
}
