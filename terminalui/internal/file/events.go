package file

import (
	"log/slog"

	tea "charm.land/bubbletea/v2"
)

type LoadEvent string

type ContentEvent []byte

type UpdateEvent []byte

func (c ContentEvent) LogValue() slog.Value {
	return slog.StringValue("data:" + string(c[:min(len(c), 30)]) + "...")
}

type saveEvent []byte

func Save(content []byte) tea.Cmd {
	return func() tea.Msg {
		return saveEvent(content)
	}
}
