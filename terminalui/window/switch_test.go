package window_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/dkotik/cuebook/terminalui/card"
	"github.com/dkotik/cuebook/terminalui/window"
)

func TestStatePreservationWhenSwitching(t *testing.T) {
	t.Skip("impl")
	w := window.New(card.New("test"), nil)
	var cmd tea.Cmd

	w, cmd = w.Init()
	w, cmd = w.Update(window.SwitchTo(nil))

	t.Log(w, cmd)
	t.Fatal("impl")
}
