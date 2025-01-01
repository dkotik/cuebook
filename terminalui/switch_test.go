package terminalui_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/dkotik/cuebook/terminalui"
	"github.com/dkotik/cuebook/terminalui/card"
)

func TestStatePreservationWhenSwitching(t *testing.T) {
	t.Skip("impl")
	window := terminalui.New(card.New("test"), nil)
	var cmd tea.Cmd

	window, cmd = window.Init()
	window, cmd = window.Update(terminalui.SwitchTo(nil))

	t.Log(window, cmd)
	t.Fatal("impl")
}
