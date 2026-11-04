package textarea_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/dkotik/cuebook/terminalui/internal/textarea"
)

type submitted string

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		configuration textarea.Configuration
		wantErr       string
		wantSubmitted tea.Msg
	}{
		{name: "missing label", wantErr: "label is required"},
		{name: "blank label", configuration: textarea.Configuration{Label: "  "}, wantErr: "label is required"},
		{name: "nil submit command does nothing", configuration: textarea.Configuration{Label: "Notes", Value: "draft"}},
		{
			name:          "blank value is accepted",
			configuration: textarea.Configuration{Label: "Notes", Value: "   ", OnSubmit: submit},
			wantSubmitted: submitted("   "),
		},
		{
			name:          "submits current value",
			configuration: textarea.Configuration{Label: "Notes", Value: "draft", OnSubmit: submit},
			wantSubmitted: submitted("draft"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			model, err := textarea.New(tt.configuration)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("New() error = %v, want error containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			model, _ = model.Init()
			_, cmd := model.Update(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
			if tt.wantSubmitted == nil {
				if cmd != nil {
					t.Fatalf("save returned a command producing %#v, want none", cmd())
				}
				return
			}
			if cmd == nil {
				t.Fatal("save did not return a command")
			}
			if got := cmd(); got != tt.wantSubmitted {
				t.Errorf("submitted %#v, want %#v", got, tt.wantSubmitted)
			}
		})
	}
}

func TestViewStyles(t *testing.T) {
	t.Parallel()

	custom := lipgloss.NewStyle().Foreground(lipgloss.Green).Italic(true)
	frame := lipgloss.NewStyle().Border(lipgloss.ThickBorder()).BorderForeground(lipgloss.Red)
	tests := []struct {
		name          string
		configuration textarea.Configuration
		wantLabel     string
		wantBorder    string
	}{
		{
			name:          "default label and frame",
			configuration: textarea.Configuration{Label: "Notes"},
			wantLabel:     " Notes:",
			wantBorder:    lipgloss.NormalBorder().TopLeft,
		},
		{
			name:          "custom label style",
			configuration: textarea.Configuration{Label: "Notes", LabelStyle: custom},
			wantLabel:     custom.Render(" Notes:"),
			wantBorder:    lipgloss.NormalBorder().TopLeft,
		},
		{
			name:          "custom frame style",
			configuration: textarea.Configuration{Label: "Notes", FrameStyle: frame},
			wantLabel:     " Notes:",
			wantBorder:    lipgloss.ThickBorder().TopLeft,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			model, err := textarea.New(tt.configuration)
			if err != nil {
				t.Fatal(err)
			}
			model, _ = model.Update(tea.WindowSizeMsg{Width: 40, Height: 12})
			view := model.View()
			label, body, _ := strings.Cut(view, "\n")
			if strings.TrimRight(label, " ") != tt.wantLabel {
				t.Errorf("label line = %q, want %q", label, tt.wantLabel)
			}
			if !strings.Contains(body, tt.wantBorder) {
				t.Errorf("frame does not contain %q:\n%s", tt.wantBorder, body)
			}
		})
	}
}

func submit(value string) tea.Cmd {
	return func() tea.Msg { return submitted(value) }
}
