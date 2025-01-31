package entry

import (
	"os"
	"strings"
	"testing"
	"time"

	"cuelang.org/go/cue/cuecontext"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/x/exp/teatest/v2"
)

func TestCreateForm(t *testing.T) {
	large, err := os.ReadFile("../../../test/testdata/simple.cue")
	if err != nil {
		t.Fatal("unable to read test file")
	}
	book := cuecontext.New().CompileBytes(large)
	if err = book.Err(); err != nil {
		t.Fatal("unable to parse test file:", err)
	}

	tm := teatest.NewTestModel(t, createForm{})
	tm.Send(emitCreateFormFields(book))
	tm.Send(tea.WindowSizeMsg{Width: 20, Height: 20})
	waitForString(t, tm, "Quit? (y/N)")
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
