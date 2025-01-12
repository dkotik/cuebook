package window_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/dkotik/cuebook/terminalui/card"
	"github.com/dkotik/cuebook/terminalui/window"
)

func TestStatePreservationWhenSwitching(t *testing.T) {
	t.Skip("impl")
	w, err := window.New(window.WithInitialModels(card.New("test")))
	if err != nil {
		t.Fatal(err)
	}
	var cmd tea.Cmd

	w, cmd = w.Init()
	w, cmd = w.Update(window.SwitchTo(nil))

	t.Log(w, cmd)
	t.Fatal("impl")
}
