package status_test

import (
	"testing"

	"github.com/charmbracelet/bubbles/v2/help"
	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/spinner"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/dkotik/cuebook/terminalui/internal/status"
	"github.com/dkotik/cuebook/terminalui/window"
)

func TestStatusView(t *testing.T) {
	t.Parallel()

	bindings := []key.Binding{
		key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("ctrl+s", "save")),
		key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	}
	defaults := help.New().Styles
	blue := lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
	custom := lipgloss.NewStyle().Foreground(lipgloss.Green).Italic(true)

	helpView := func(keyStyle, description, separator lipgloss.Style) string {
		sep := separator.Inline(true).Render(" • ")
		item := func(b key.Binding) string {
			return keyStyle.Inline(true).Render(b.Help().Key) + " " + description.Inline(true).Render(b.Help().Desc)
		}
		return item(bindings[0]) + sep + item(bindings[1])
	}
	spinnerFrame := func(style lipgloss.Style) string {
		return style.Render(spinner.MiniDot.Frames[0])
	}

	tests := []struct {
		name          string
		configuration status.Configuration
		busy          bool
		want          string
	}{
		{
			name:          "idle without bindings",
			configuration: status.Configuration{},
			want:          "   false",
		},
		{
			name:          "idle default help styles",
			configuration: status.Configuration{KeyBindings: bindings},
			want:          "  " + helpView(defaults.ShortKey, defaults.ShortDesc, defaults.ShortSeparator) + " false",
		},
		{
			name:          "busy default spinner",
			configuration: status.Configuration{KeyBindings: bindings},
			busy:          true,
			want:          spinnerFrame(blue) + " " + helpView(defaults.ShortKey, defaults.ShortDesc, defaults.ShortSeparator) + " true",
		},
		{
			name: "custom styles",
			configuration: status.Configuration{
				KeyBindings:      bindings,
				SpinnerStyle:     custom,
				KeyStyle:         custom,
				DescriptionStyle: custom,
				SeparatorStyle:   custom,
			},
			busy: true,
			want: spinnerFrame(custom) + " " + helpView(custom, custom, custom) + " true",
		},
		{
			name: "partial spinner style inherits default color",
			configuration: status.Configuration{
				SpinnerStyle: lipgloss.NewStyle().Bold(true),
			},
			busy: true,
			want: spinnerFrame(blue.Bold(true)) + "  true",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			model := status.New(tt.configuration)
			model, _ = model.Update(window.IsBusyEvent(tt.busy))
			if got := model.View(); got != tt.want {
				t.Errorf("View() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStatusStartsSpinnerOnlyWhenBusy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		busy     bool
		wantTick bool
	}{
		{name: "busy starts spinner", busy: true, wantTick: true},
		{name: "idle does not start spinner"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, cmd := status.New(status.Configuration{}).Update(window.IsBusyEvent(tt.busy))
			if (cmd != nil) != tt.wantTick {
				t.Fatalf("command returned = %t, want %t", cmd != nil, tt.wantTick)
			}
			if cmd == nil {
				return
			}
			if _, ok := cmd().(spinner.TickMsg); !ok {
				t.Errorf("command did not return a spinner tick")
			}
		})
	}
}
