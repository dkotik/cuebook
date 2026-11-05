package internal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/exp/teatest/v2"
	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/terminalui/event"
	"github.com/dkotik/cuebook/terminalui/internal/file"
	"github.com/dkotik/cuebook/terminalui/window"
)

func TestAddFieldThroughTheTerminalUI(t *testing.T) {
	source := []byte(`[{Name: "Ada"}]`)
	book, err := cuebook.New(source)
	if err != nil {
		t.Fatal(err)
	}
	ui, err := window.New(window.WithInitialModels(NewEntryList()), WithStateEventTransformers())
	if err != nil {
		t.Fatal(err)
	}
	tm := teatest.NewTestModel(t, ui, teatest.WithInitialTermSize(termWidth, termHeight))
	tm.Send(cuebook.Result{Document: book, Source: source, Hash: 1})
	time.Sleep(50 * time.Millisecond)
	tm.Send(tea.WindowSizeMsg{Width: termWidth, Height: termHeight})
	waitFor(t, tm, "Ada")

	press(tm, tea.KeyPressMsg{Code: tea.KeyDown}) // title card -> entry
	press(tm, tea.KeyPressMsg{Code: tea.KeyEnter})
	waitFor(t, tm, "Add a Custom Field")
	press(tm, tea.KeyPressMsg{Code: tea.KeyDown}) // Name -> Add a Custom Field
	press(tm, tea.KeyPressMsg{Code: tea.KeyEnter})
	waitFor(t, tm, "Type")
	tm.Type("Role")
	press(tm, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	time.Sleep(time.Second)

	// A panic in the program surfaces only when it stops.
	if err := tm.Quit(); err != nil {
		t.Fatal(err)
	}
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

// press sends a key and then the terminal size, which a real terminal reports
// in answer to the size requests that new screens make.
func TestOpenNewFieldFormLikeTheApp(t *testing.T) {
	source, err := os.ReadFile("../../test/testdata/simple.cue")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "simple.cue")
	if err := os.WriteFile(path, source, 0o600); err != nil {
		t.Fatal(err)
	}
	picker, err := file.New(file.Configuration{AllowedExtensions: []string{".cue"}})
	if err != nil {
		t.Fatal(err)
	}
	ui, err := window.New(
		window.WithInitialModels(event.NewDecorator(ParseFileToBookAndCreateEntryListIfNeeded)(picker)),
		WithStateEventTransformers(),
	)
	if err != nil {
		t.Fatal(err)
	}
	tm := teatest.NewTestModel(t, ui, teatest.WithInitialTermSize(termWidth, termHeight))
	tm.Send(file.LoadEvent(path))
	time.Sleep(100 * time.Millisecond)
	tm.Send(tea.WindowSizeMsg{Width: termWidth, Height: termHeight})
	waitFor(t, tm, "First11111aa")

	press(tm, tea.KeyPressMsg{Code: tea.KeyDown})
	press(tm, tea.KeyPressMsg{Code: tea.KeyEnter})
	for range 6 {
		press(tm, tea.KeyPressMsg{Code: tea.KeyDown})
	}
	press(tm, tea.KeyPressMsg{Code: tea.KeyUp})
	press(tm, tea.KeyPressMsg{Code: tea.KeyUp})
	press(tm, tea.KeyPressMsg{Code: tea.KeyEnter})
	time.Sleep(500 * time.Millisecond)

	if err := tm.Quit(); err != nil {
		t.Fatal(err)
	}
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func press(tm *teatest.TestModel, key tea.KeyPressMsg) {
	tm.Send(key)
	time.Sleep(50 * time.Millisecond)
	tm.Send(tea.WindowSizeMsg{Width: termWidth, Height: termHeight})
	time.Sleep(50 * time.Millisecond)
}

// The size of a default terminal window.
const termWidth, termHeight = 80, 24

func waitFor(t *testing.T, tm *teatest.TestModel, text string) {
	t.Helper()
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return strings.Contains(string(b), text)
	}, teatest.WithCheckInterval(20*time.Millisecond), teatest.WithDuration(3*time.Second))
}
