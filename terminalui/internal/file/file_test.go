package file

import (
	"slices"
	"strings"
	"testing"

	"charm.land/bubbles/v2/filepicker"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func TestNewValidatesAllowedExtensions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		extensions []string
		want       []string
		wantErr    string
	}{
		{name: "missing", wantErr: "no file extensions provided"},
		{name: "empty", extensions: []string{""}, wantErr: "empty file extension provided"},
		{name: "missing dot", extensions: []string{"cue"}, wantErr: "must begin with a dot"},
		{name: "single", extensions: []string{".cue"}, want: []string{".cue"}},
		{name: "duplicates removed", extensions: []string{".cue", ".json", ".cue"}, want: []string{".cue", ".json"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			model, err := New(Configuration{AllowedExtensions: tt.extensions})
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("New() error = %v, want error containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got := model.(file).AllowedTypes; !slices.Equal(got, tt.want) {
				t.Errorf("AllowedTypes = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewSelectionCommand(t *testing.T) {
	t.Parallel()

	type customSelection string
	tests := []struct {
		name     string
		onSelect Command
		want     tea.Msg
	}{
		{name: "default emits load event", want: LoadEvent("book.cue")},
		{
			name:     "custom command",
			onSelect: func(path string) tea.Cmd { return func() tea.Msg { return customSelection(path) } },
			want:     customSelection("book.cue"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			model, err := New(Configuration{AllowedExtensions: []string{".cue"}, OnSelect: tt.onSelect})
			if err != nil {
				t.Fatal(err)
			}
			if got := model.(file).OnSelect("book.cue")(); got != tt.want {
				t.Errorf("OnSelect() message = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestNewStyles(t *testing.T) {
	t.Parallel()

	defaults := filepicker.DefaultStyles()
	custom := lipgloss.NewStyle().Foreground(lipgloss.Green).Italic(true)
	tests := []struct {
		name          string
		configuration Configuration
		got           func(filepicker.Styles) lipgloss.Style
		want          lipgloss.Style
		text          string
	}{
		{name: "default cursor", got: func(s filepicker.Styles) lipgloss.Style { return s.Cursor }, want: defaults.Cursor, text: ">"},
		{name: "default selected", got: func(s filepicker.Styles) lipgloss.Style { return s.Selected }, want: defaults.Selected, text: "book.cue"},
		{
			name:          "custom directory",
			configuration: Configuration{DirectoryStyle: custom},
			got:           func(s filepicker.Styles) lipgloss.Style { return s.Directory },
			want:          custom,
			text:          "notes",
		},
		{
			name:          "partial selected style keeps default color and bold",
			configuration: Configuration{SelectedStyle: lipgloss.NewStyle().Underline(true)},
			got:           func(s filepicker.Styles) lipgloss.Style { return s.Selected },
			want:          defaults.Selected.Underline(true),
			text:          "book.cue",
		},
		{name: "default empty directory", got: func(s filepicker.Styles) lipgloss.Style { return s.EmptyDirectory }, want: defaults.EmptyDirectory},
		{
			name:          "custom empty directory keeps default padding and message",
			configuration: Configuration{EmptyDirectoryStyle: custom},
			got:           func(s filepicker.Styles) lipgloss.Style { return s.EmptyDirectory },
			want:          custom.PaddingLeft(defaults.EmptyDirectory.GetPaddingLeft()).SetString(defaults.EmptyDirectory.Value()),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.configuration.AllowedExtensions = []string{".cue"}
			model, err := New(tt.configuration)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := tt.got(model.(file).Styles).Render(tt.text), tt.want.Render(tt.text); got != want {
				t.Errorf("Render() = %q, want %q", got, want)
			}
		})
	}
}
