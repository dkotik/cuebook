package markdown_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/styles"
	"github.com/dkotik/cuebook/terminalui/internal/markdown"
)

func TestNew(t *testing.T) {
	t.Parallel()

	const text = "# Title\n\nSome *emphasis* here."
	custom := styles.ASCIIStyleConfig
	tests := []struct {
		name          string
		configuration markdown.Configuration
		wantStyle     string
		wantErr       string
	}{
		{name: "unknown style", configuration: markdown.Configuration{Text: text, Style: "neon"}, wantErr: `unknown Markdown style "neon"`},
		{name: "named dark style", configuration: markdown.Configuration{Text: text, Style: styles.DarkStyle}, wantStyle: styles.DarkStyle},
		{name: "named light style", configuration: markdown.Configuration{Text: text, Style: styles.LightStyle}, wantStyle: styles.LightStyle},
		{name: "named ascii style", configuration: markdown.Configuration{Text: text, Style: styles.AsciiStyle}, wantStyle: styles.AsciiStyle},
		{
			name:          "style config overrides name",
			configuration: markdown.Configuration{Text: text, Style: "neon", StyleConfig: &custom},
			wantStyle:     styles.AsciiStyle,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			model, err := markdown.New(tt.configuration)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("New() error = %v, want error containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			model, _ = model.Update(tea.WindowSizeMsg{Width: 42, Height: 10})

			renderer, err := glamour.NewTermRenderer(
				glamour.WithWordWrap(40),
				glamour.WithStandardStyle(tt.wantStyle),
			)
			if err != nil {
				t.Fatal(err)
			}
			want, err := renderer.Render(text)
			if err != nil {
				t.Fatal(err)
			}
			if got := model.View(); got != want {
				t.Errorf("View() =\n%q\nwant\n%q", got, want)
			}
		})
	}
}
