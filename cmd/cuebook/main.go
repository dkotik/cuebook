package main

import (
	"context"
	"os"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dkotik/cuebook/terminalui"
	"github.com/urfave/cli/v3"
)

var (
	titleStyle      = lipgloss.NewStyle().MarginLeft(2)
	paginationStyle = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle       = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
)

func main() {
	cmd := &cli.Command{
		Name:  "cuebook",
		Usage: "edit lists of structured data items",
		Action: func(context.Context, *cli.Command) (err error) {
			items := []list.Item{
				terminalui.Item("Ramen"),
				terminalui.Item("Tomato Soup"),
				terminalui.Item("Hamburgers"),
				terminalui.Item("Cheeseburgers"),
				terminalui.Item("Currywurst"),
				terminalui.Item("Okonomiyaki"),
				terminalui.Item("Pasta"),
				terminalui.Item("Fillet Mignon"),
				terminalui.Item("Caviar"),
				terminalui.Item("Just Wine"),
			}

			const defaultWidth = 20

			l := list.New(items, terminalui.ItemDelegate{}, defaultWidth, 14)
			l.Title = "What do you want for dinner?"
			l.SetShowStatusBar(false)
			l.SetFilteringEnabled(false)
			l.Styles.Title = titleStyle
			l.Styles.PaginationStyle = paginationStyle
			l.Styles.HelpStyle = helpStyle

			_, err = tea.NewProgram(terminalui.List{List: l}).Run()
			return err
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		panic(err)
	}
}
