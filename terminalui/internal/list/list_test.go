package list_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/dkotik/cuebook/terminalui/internal/list"
)

func TestListView(t *testing.T) {
	t.Parallel()

	size := tea.WindowSizeMsg{Width: 40, Height: 5}
	layout := func(style lipgloss.Style) lipgloss.Style {
		return style.Height(size.Height).MarginLeft(size.Width / 4)
	}
	centered := lipgloss.NewStyle().AlignVertical(lipgloss.Center)
	custom := lipgloss.NewStyle().Foreground(lipgloss.Green)

	tests := []struct {
		name          string
		configuration list.Configuration
		want          string
	}{
		{
			name: "empty list",
			want: "<empty list>",
		},
		{
			name:          "default style centers vertically",
			configuration: list.Configuration{Items: []tea.Model{item("one")}},
			want:          layout(centered).Render("one"),
		},
		{
			name: "custom style inherits default centering",
			configuration: list.Configuration{
				Items: []tea.Model{item("one")},
				Style: custom,
			},
			want: layout(custom.AlignVertical(lipgloss.Center)).Render("one"),
		},
		{
			name: "custom alignment overrides default",
			configuration: list.Configuration{
				Items: []tea.Model{item("one")},
				Style: lipgloss.NewStyle().AlignVertical(lipgloss.Top),
			},
			want: layout(lipgloss.NewStyle().AlignVertical(lipgloss.Top)).Render("one"),
		},
		{
			name: "list manages height and margin",
			configuration: list.Configuration{
				Items: []tea.Model{item("one")},
				Style: lipgloss.NewStyle().Height(1).MarginLeft(0),
			},
			want: layout(centered).Render("one"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			model := list.New(tt.configuration)
			model, _ = model.Init()
			model, _ = model.Update(size)
			if got := model.View(); got != tt.want {
				t.Errorf("View() =\n%q\nwant\n%q", got, tt.want)
			}
		})
	}
}

func TestListHighlightsSelectedItem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		keys      []tea.KeyPressMsg
		wantIndex int
	}{
		{name: "first item highlighted on init", wantIndex: 0},
		{name: "tab moves forward", keys: []tea.KeyPressMsg{{Code: tea.KeyTab}}, wantIndex: 1},
		{name: "up wraps to last item", keys: []tea.KeyPressMsg{{Code: tea.KeyUp}}, wantIndex: 2},
		{name: "down then shift+tab returns", keys: []tea.KeyPressMsg{{Code: tea.KeyDown}, {Code: tea.KeyTab, Mod: tea.ModShift}}, wantIndex: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			items := []*highlightItem{{}, {}, {}}
			models := make([]tea.Model, len(items))
			for i, it := range items {
				models[i] = it
			}
			model := list.New(list.Configuration{Items: models})
			model, _ = model.Init()
			model, _ = model.Update(tea.WindowSizeMsg{Width: 40, Height: 20})
			for _, key := range tt.keys {
				model, _ = model.Update(key)
			}
			for i, it := range items {
				if want := i == tt.wantIndex; it.highlighted != want {
					t.Errorf("item %d highlighted = %t, want %t", i, it.highlighted, want)
				}
			}
		})
	}
}

type item string

func (i item) Init() (tea.Model, tea.Cmd)          { return i, nil }
func (i item) Update(tea.Msg) (tea.Model, tea.Cmd) { return i, nil }
func (i item) View() string                        { return string(i) }

type highlightItem struct {
	highlighted bool
}

func (h *highlightItem) Init() (tea.Model, tea.Cmd) { return h, nil }

func (h *highlightItem) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if highlighted, ok := msg.(list.HighlightHintEvent); ok {
		h.highlighted = bool(highlighted)
	}
	return h, nil
}

func (*highlightItem) View() string { return "item" }
