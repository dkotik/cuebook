package list

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/x/exp/teatest/v2"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func TestCreateForm(t *testing.T) {
	button := NewButton(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "testButton",
			Other: "Test Button",
		},
	}, nil)

	tm := teatest.NewTestModel(t, button)
	// tm.Send(emitCreateFormFields(book))
	tm.Send(tea.WindowSizeMsg{Width: 20, Height: 20})
	waitForString(t, tm, "Button")
	// tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
	t.Log("test model", tm)
}

func waitForString(t *testing.T, tm *teatest.TestModel, s string) {
	teatest.WaitFor(
		t,
		tm.Output(),
		func(b []byte) bool {
			return strings.Contains(string(b), s)
		},
		teatest.WithCheckInterval(time.Millisecond*100),
		teatest.WithDuration(time.Second),
	)
}
