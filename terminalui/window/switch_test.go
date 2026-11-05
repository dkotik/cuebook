package window_test

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/dkotik/cuebook/terminalui/internal/card"
	"github.com/dkotik/cuebook/terminalui/window"
)

func TestStatePreservationWhenSwitching(t *testing.T) {
	t.Skip("impl")
	w, err := window.New(window.WithInitialModels(card.New(card.Configuration{Title: "test"})))
	if err != nil {
		t.Fatal(err)
	}
	var cmd tea.Cmd

	cmd = w.Init()
	w, cmd = w.Update(window.SwitchTo(nil))

	t.Log(w, cmd)
	t.Fatal("impl")
}
