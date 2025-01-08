package internal

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/dkotik/cuebook"
)

func parseBook(source []byte) tea.Cmd {
	return func() tea.Msg {
		book, err := cuebook.New(source)
		if err != nil {
			return err
		}
		return parsedBook{
			Book:   book,
			Source: source,
		}
	}
}
