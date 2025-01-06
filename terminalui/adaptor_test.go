package terminalui_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/dkotik/cuebook/terminalui"
	"github.com/dkotik/cuebook/terminalui/card"
)

func TestUpdateAdaptor(t *testing.T) {
	type testEvent string

	card := card.New("title")
	adapted := terminalui.NewEventAdaptor(func(m tea.Model, e testEvent) (tea.Model, tea.Cmd) {
		return m, func() tea.Msg {
			return 7
		}
	})(card)
	adapted, _ = adapted.Init()

	adapted, cmd := adapted.Update(testEvent("test"))
	if cmd() != 7 {
		t.Fatal("update was not triggered")
	}
}
