package card_test

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/dkotik/cuebook/terminalui/internal/card"
	"github.com/dkotik/cuebook/terminalui/internal/list"
)

func TestCardView(t *testing.T) {
	t.Parallel()

	const width = 20
	defaultTitle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
	inactive := lipgloss.NewStyle().Width(width - 3).PaddingLeft(2)
	active := lipgloss.NewStyle().Width(width - 3).PaddingLeft(1).
		BorderLeft(true).BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("65"))
	customTitle := lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Magenta)
	customDescription := lipgloss.NewStyle().Foreground(lipgloss.Yellow)
	customActive := lipgloss.NewStyle().BorderLeft(true).BorderStyle(lipgloss.ThickBorder()).BorderForeground(lipgloss.Red)

	lines := func(title lipgloss.Style, body lipgloss.Style, titleText string, description ...string) string {
		out := []string{body.Inherit(title).Render(titleText)}
		for _, line := range description {
			out = append(out, body.Render(line))
		}
		return strings.Join(out, "\n")
	}

	tests := []struct {
		name          string
		configuration card.Configuration
		selected      bool
		width         int
		want          string
	}{
		{
			name:          "narrow placeholder",
			configuration: card.Configuration{Title: "Ada"},
			width:         3,
			want:          "</>",
		},
		{
			name:          "blank title falls back",
			configuration: card.Configuration{Title: "  "},
			width:         width,
			want:          lines(defaultTitle, inactive, "???"),
		},
		{
			name:          "inactive default with blank description lines removed",
			configuration: card.Configuration{Title: " Ada ", Description: []string{"one", " ", "two"}},
			width:         width,
			want:          lines(defaultTitle, inactive, "Ada", "one", "two"),
		},
		{
			name:          "active default border",
			configuration: card.Configuration{Title: "Ada", Description: []string{"one"}},
			selected:      true,
			width:         width,
			want:          lines(defaultTitle, active, "Ada", "one"),
		},
		{
			name: "custom styles",
			configuration: card.Configuration{
				Title:            "Ada",
				Description:      []string{"one"},
				TitleStyle:       customTitle,
				DescriptionStyle: customDescription,
				ActiveStyle:      customActive,
			},
			selected: true,
			width:    width,
			want: func() string {
				body := lipgloss.NewStyle().Width(width - 3).PaddingLeft(1).Inherit(customActive)
				// Bold is unset in customTitle, so it is inherited from the default.
				return body.Inherit(customTitle.Bold(true)).Render("Ada") + "\n" + body.Inherit(customDescription).Render("one")
			}(),
		},
		{
			name: "partial title style inherits default color",
			configuration: card.Configuration{
				Title:      "Ada",
				TitleStyle: lipgloss.NewStyle().Underline(true),
			},
			width: width,
			want:  lines(defaultTitle.Underline(true), inactive, "Ada"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			model := card.New(tt.configuration)
			model, _ = model.Update(tea.WindowSizeMsg{Width: tt.width, Height: 10})
			model, _ = model.Update(list.HighlightHintEvent(tt.selected))
			if got := model.View().Content; got != tt.want {
				t.Errorf("View() =\n%q\nwant\n%q", got, tt.want)
			}
		})
	}
}

func TestNewDoesNotModifyDescription(t *testing.T) {
	t.Parallel()

	description := []string{"one", "", "two"}
	card.New(card.Configuration{Title: "Ada", Description: description})
	if got := strings.Join(description, ","); got != "one,,two" {
		t.Fatalf("description = %q, want original slice unchanged", got)
	}
}
