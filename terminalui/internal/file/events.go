package file

import (
	"log/slog"

	tea "charm.land/bubbletea/v2"
)

type LoadEvent string

type ContentEvent struct {
	Path    string
	Content []byte
}

type UpdateEvent struct {
	Path    string
	Content []byte
}

func (c ContentEvent) LogValue() slog.Value {
	return slog.StringValue("path:" + c.Path + " data:" + string(c.Content[:min(len(c.Content), 30)]) + "...")
}

type saveEvent []byte

func Save(content []byte) tea.Cmd {
	return func() tea.Msg {
		return saveEvent(content)
	}
}
