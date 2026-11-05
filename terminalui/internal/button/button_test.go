package button_test

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/dkotik/cuebook/terminalui/internal/button"
	"github.com/dkotik/cuebook/terminalui/internal/list"
)

func TestButtonView(t *testing.T) {
	t.Parallel()

	gray := lipgloss.NewStyle().Foreground(lipgloss.Color("#999999"))
	teal := lipgloss.NewStyle().Foreground(lipgloss.Color("73"))
	customInactive := lipgloss.NewStyle().Foreground(lipgloss.Green).Italic(true)
	customActive := lipgloss.NewStyle().Foreground(lipgloss.Magenta).Bold(true)

	tests := []struct {
		name          string
		configuration button.Configuration
		highlights    []bool
		want          string
	}{
		{
			name:          "inactive default",
			configuration: button.Configuration{Text: "Save"},
			want:          gray.Render("Save"),
		},
		{
			name:          "active default",
			configuration: button.Configuration{Text: "Save"},
			highlights:    []bool{true},
			want:          teal.Render("Save"),
		},
		{
			name:          "deselection restores inactive style",
			configuration: button.Configuration{Text: "Save"},
			highlights:    []bool{true, false},
			want:          gray.Render("Save"),
		},
		{
			name:          "custom inactive style",
			configuration: button.Configuration{Text: "Save", InactiveStyle: customInactive, ActiveStyle: customActive},
			want:          customInactive.Render("Save"),
		},
		{
			name:          "custom active style",
			configuration: button.Configuration{Text: "Save", InactiveStyle: customInactive, ActiveStyle: customActive},
			highlights:    []bool{true},
			want:          customActive.Render("Save"),
		},
		{
			name:          "partial style inherits default color",
			configuration: button.Configuration{Text: "Save", InactiveStyle: lipgloss.NewStyle().Underline(true)},
			want:          gray.Underline(true).Render("Save"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			model := button.New(tt.configuration)
			model, _ = model.Update(tea.WindowSizeMsg{Width: 40, Height: 10})
			for _, highlighted := range tt.highlights {
				model, _ = model.Update(list.HighlightHintEvent(highlighted))
			}
			if got := model.View().Content; got != tt.want {
				t.Errorf("View() = %q, want %q", got, tt.want)
			}
		})
	}
}
