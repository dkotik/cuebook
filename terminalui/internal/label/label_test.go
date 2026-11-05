package label_test

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/dkotik/cuebook/terminalui/internal/label"
	"github.com/dkotik/cuebook/terminalui/internal/list"
)

func TestLabelStyles(t *testing.T) {
	t.Parallel()

	gray := lipgloss.NewStyle().Foreground(lipgloss.Color(label.DefaultInactiveColor))
	blue := lipgloss.NewStyle().Foreground(lipgloss.Blue)
	customInactive := lipgloss.NewStyle().Foreground(lipgloss.Green).Italic(true)
	customActive := lipgloss.NewStyle().Foreground(lipgloss.Magenta).Bold(true)
	tests := []struct {
		name          string
		configuration *label.Configuration
		highlights    []bool
		wantLabel     string
	}{
		{
			name:          "blank default configuration",
			configuration: nil,
			wantLabel:     gray.Render(""),
		},
		{
			name:          "inactive defaults to gray",
			configuration: &label.Configuration{Text: "Name"},
			wantLabel:     gray.Render("Name:"),
		},
		{
			name:          "active retains blue default",
			configuration: &label.Configuration{Text: "Name"},
			highlights:    []bool{true},
			wantLabel:     blue.Render("Name:"),
		},
		{
			name:          "deselection restores gray",
			configuration: &label.Configuration{Text: "Name"},
			highlights:    []bool{true, false},
			wantLabel:     gray.Render("Name:"),
		},
		{
			name: "custom inactive style",
			configuration: &label.Configuration{
				Text: "Name", InactiveStyle: customInactive, ActiveStyle: customActive,
			},
			wantLabel: customInactive.Render("Name:"),
		},
		{
			name: "custom active style",
			configuration: &label.Configuration{
				Text: "Name", InactiveStyle: customInactive, ActiveStyle: customActive,
			},
			highlights: []bool{true},
			wantLabel:  customActive.Render("Name:"),
		},
		{
			name: "partial style inherits default foreground",
			configuration: &label.Configuration{
				Text: "Name", InactiveStyle: lipgloss.NewStyle().Bold(true),
			},
			wantLabel: gray.Bold(true).Render("Name:"),
		},
		{
			name:          "default required marker",
			configuration: &label.Configuration{Text: "Name", Required: true},
			wantLabel:     lipgloss.NewStyle().Foreground(lipgloss.BrightRed).Render("✱") + gray.Render("Name:"),
		},
		{
			name: "custom required marker",
			configuration: &label.Configuration{
				Text: "Name", Required: true, RequiredMarkerStyle: customActive,
			},
			wantLabel: customActive.Render("✱") + gray.Render("Name:"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			wrapped := &recordingModel{}
			model := label.New(wrapped, tt.configuration)
			for _, highlighted := range tt.highlights {
				model, _ = model.Update(list.HighlightHintEvent(highlighted))
			}
			want := lipgloss.JoinHorizontal(lipgloss.Left, tt.wantLabel, wrapped.View().Content)
			if got := model.View().Content; got != want {
				t.Errorf("View() = %q, want %q", got, want)
			}
			if len(wrapped.messages) != len(tt.highlights) {
				t.Errorf("wrapped model received %d messages, want %d highlight events", len(wrapped.messages), len(tt.highlights))
			}
		})
	}
}

func TestResponsiveLayout(t *testing.T) {
	t.Parallel()

	horizontal := label.LayoutChangeEvent{Width: 10, IsHorizontal: true}
	vertical := label.LayoutChangeEvent{IsHorizontal: false}
	tests := []struct {
		name           string
		text           string
		active         bool
		layouts        []label.LayoutChangeEvent
		wantHorizontal bool
		wantWidth      int
	}{
		{name: "horizontal", text: "Name", layouts: []label.LayoutChangeEvent{horizontal}, wantHorizontal: true, wantWidth: 20},
		{name: "vertical", text: "Name", layouts: []label.LayoutChangeEvent{vertical}, wantWidth: 30},
		{name: "horizontal to vertical retains active style", text: "Name", active: true, layouts: []label.LayoutChangeEvent{horizontal, vertical}, wantWidth: 30},
		{name: "vertical to horizontal retains active style", text: "Name", active: true, layouts: []label.LayoutChangeEvent{vertical, horizontal}, wantHorizontal: true, wantWidth: 20},
		{name: "blank horizontal reserves label column", layouts: []label.LayoutChangeEvent{horizontal}, wantHorizontal: true, wantWidth: 20},
		{name: "blank vertical omits label row", layouts: []label.LayoutChangeEvent{horizontal, vertical}, wantWidth: 30},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			inactive := lipgloss.NewStyle().Foreground(lipgloss.Green)
			active := lipgloss.NewStyle().Foreground(lipgloss.Magenta)
			wrapped := &recordingModel{}
			model := label.New(wrapped, &label.Configuration{
				Text: tt.text, InactiveStyle: inactive, ActiveStyle: active,
			})
			model, _ = model.Update(list.HighlightHintEvent(tt.active))
			for _, layout := range tt.layouts {
				model, _ = model.Update(layout)
			}
			model, _ = model.Update(tea.WindowSizeMsg{Width: 30, Height: 8})
			last := wrapped.messages[len(wrapped.messages)-1]
			if got, ok := last.(tea.WindowSizeMsg); !ok || got.Width != tt.wantWidth || got.Height != 8 {
				t.Errorf("wrapped model size = %#v, want width %d and height 8", last, tt.wantWidth)
			}

			style := inactive
			if tt.active && tt.text != "" {
				style = active
			}
			text := tt.text
			if text != "" {
				text += ":"
			}
			want := wrapped.View().Content
			if tt.wantHorizontal {
				style = style.Align(lipgloss.Right).PaddingRight(1).Width(10)
				want = lipgloss.JoinHorizontal(lipgloss.Left, style.Render(text), want)
			} else if text != "" {
				want = lipgloss.JoinVertical(lipgloss.Left, style.Render(text), want)
			}
			if got := model.View().Content; got != want {
				t.Errorf("View() = %q, want %q", got, want)
			}
		})
	}
}

type recordingModel struct {
	messages []tea.Msg
}

func (m *recordingModel) Init() tea.Cmd { return nil }

func (m *recordingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.messages = append(m.messages, msg)
	return m, nil
}

func (*recordingModel) View() tea.View { return tea.NewView("value") }
