package event_test

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/dkotik/cuebook/terminalui/event"
	"github.com/dkotik/cuebook/terminalui/internal/card"
)

func TestUpdateAdaptor(t *testing.T) {
	type testEvent string

	card := card.New(card.Configuration{Title: "title"})
	adapted := event.NewAdaptor(func(m tea.Model, e testEvent) (tea.Model, tea.Cmd) {
		return m, func() tea.Msg {
			return 7
		}
	})(card)
	_ = adapted.Init()

	adapted, cmd := adapted.Update(testEvent("test"))
	if cmd() != 7 {
		t.Fatal("update was not triggered")
	}
}
