package window_test

import (
	"reflect"
	"testing"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/dkotik/cuebook/terminalui/window"
)

func TestTerminalShortcuts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		msg      tea.Msg
		command  tea.Cmd
		consumed bool
	}{
		{name: "Ctrl+C quits", msg: tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}, command: tea.Quit, consumed: true},
		{name: "Ctrl+backslash quits", msg: tea.KeyPressMsg{Code: '\\', Mod: tea.ModCtrl}, command: tea.Quit, consumed: true},
		{name: "Ctrl+Z suspends", msg: tea.KeyPressMsg{Code: 'z', Mod: tea.ModCtrl}, command: tea.Suspend, consumed: true},
		{name: "Ctrl+L redraws", msg: tea.KeyPressMsg{Code: 'l', Mod: tea.ModCtrl}, command: tea.ClearScreen, consumed: true},
		{name: "Ctrl+C release is ignored", msg: tea.KeyReleaseMsg{Code: 'c', Mod: tea.ModCtrl}, consumed: true},
		{name: "Ctrl+backslash release is ignored", msg: tea.KeyReleaseMsg{Code: '\\', Mod: tea.ModCtrl}, consumed: true},
		{name: "Ctrl+Z release is ignored", msg: tea.KeyReleaseMsg{Code: 'z', Mod: tea.ModCtrl}, consumed: true},
		{name: "Ctrl+L release is ignored", msg: tea.KeyReleaseMsg{Code: 'l', Mod: tea.ModCtrl}, consumed: true},
		{name: "plain C is forwarded", msg: tea.KeyPressMsg{Code: 'c'}},
		{name: "plain Z is forwarded", msg: tea.KeyPressMsg{Code: 'z'}},
		{name: "Alt+C is forwarded", msg: tea.KeyPressMsg{Code: 'c', Mod: tea.ModAlt}},
		{name: "Ctrl+Alt+C is forwarded", msg: tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl | tea.ModAlt}},
		{name: "Ctrl+Shift+C is forwarded", msg: tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl | tea.ModShift}},
		{name: "Ctrl+S is forwarded", msg: tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl}},
		{name: "Ctrl+D remains available for editing", msg: tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl}},
	}
	states := []struct {
		name string
		busy bool
	}{
		{name: "idle"},
		{name: "busy", busy: true},
	}
	for _, state := range states {
		t.Run(state.name, func(t *testing.T) {
			t.Parallel()
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					current := &shortcutModel{}
					parent := &shortcutModel{}
					watcher := &shortcutModel{}
					model, err := window.New(
						window.WithInitialModels(parent, current),
						window.WithWatchers(watcher),
					)
					if err != nil {
						t.Fatal(err)
					}
					model, _ = model.Init()
					if state.busy {
						model, _ = model.Update(window.SetBusyEvent(true))
					}

					_, cmd := model.Update(tt.msg)
					if tt.command != nil {
						if cmd == nil {
							t.Fatal("shortcut did not return a command")
						}
						// Inspect the message only; executing a SuspendMsg in a program
						// would suspend the test process group.
						got, want := cmd(), tt.command()
						if reflect.TypeOf(got) != reflect.TypeOf(want) {
							t.Errorf("shortcut returned %T, want %T", got, want)
						}
					} else if tt.consumed && cmd != nil {
						t.Error("key release should not issue a command")
					}

					wantCalls := 0
					if !tt.consumed && !state.busy {
						wantCalls = 1
					}
					if len(current.keys) != wantCalls {
						t.Errorf("current model received %d key events, want %d", len(current.keys), wantCalls)
					}
					if len(watcher.keys) != wantCalls {
						t.Errorf("watcher received %d key events, want %d", len(watcher.keys), wantCalls)
					}
					if len(parent.keys) != 0 {
						t.Error("key event unexpectedly reached a parent model")
					}
				})
			}
		})
	}
}

type shortcutModel struct {
	keys []tea.KeyMsg
}

type childHandledKey struct{}

func (m *shortcutModel) Init() (tea.Model, tea.Cmd) { return m, nil }

func (m *shortcutModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		m.keys = append(m.keys, key)
		return m, func() tea.Msg { return childHandledKey{} }
	}
	return m, nil
}

func (*shortcutModel) View() string { return "" }
