package file

import tea "github.com/charmbracelet/bubbletea/v2"

type LoadEvent string

type ContentEvent []byte

type saveEvent []byte

func Save(content []byte) tea.Cmd {
	return func() tea.Msg {
		return saveEvent(content)
	}
}
