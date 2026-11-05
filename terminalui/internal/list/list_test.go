package list_test

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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
			_ = model.Init()
			model, _ = model.Update(size)
			if got := model.View().Content; got != tt.want {
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
			_ = model.Init()
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

func TestListSwapOrder(t *testing.T) {
	t.Parallel()

	type customSwap list.SwapOrderEvent
	ctrlJ := tea.KeyPressMsg{Code: 'j', Mod: tea.ModCtrl}
	ctrlK := tea.KeyPressMsg{Code: 'k', Mod: tea.ModCtrl}
	tests := []struct {
		name        string
		onSwapOrder func(list.SwapOrderEvent) tea.Msg
		selected    int
		key         tea.KeyPressMsg
		want        tea.Msg // nil when the key must not request a swap
	}{
		{name: "ctrl+j requests moving down", selected: 1, key: ctrlJ, want: list.SwapOrderEvent{CurrentIndex: 1, DesiredIndex: 2}},
		{name: "ctrl+k requests moving up", selected: 1, key: ctrlK, want: list.SwapOrderEvent{CurrentIndex: 1, DesiredIndex: 0}},
		{name: "ctrl+down requests moving down", key: tea.KeyPressMsg{Code: tea.KeyDown, Mod: tea.ModCtrl}, want: list.SwapOrderEvent{CurrentIndex: 0, DesiredIndex: 1}},
		{name: "ctrl+up requests moving up", selected: 2, key: tea.KeyPressMsg{Code: tea.KeyUp, Mod: tea.ModCtrl}, want: list.SwapOrderEvent{CurrentIndex: 2, DesiredIndex: 1}},
		{name: "ctrl+k on the first item does nothing", key: ctrlK},
		{name: "ctrl+j on the last item does nothing", selected: 2, key: ctrlJ},
		{
			name:        "custom swap message",
			onSwapOrder: func(e list.SwapOrderEvent) tea.Msg { return customSwap(e) },
			key:         ctrlJ,
			want:        customSwap{CurrentIndex: 0, DesiredIndex: 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			model := list.New(list.Configuration{
				Items:       []tea.Model{item("a"), item("b"), item("c")},
				OnSwapOrder: tt.onSwapOrder,
			})
			_ = model.Init()
			model, _ = model.Update(list.ApplySelection(tt.selected)())
			_, cmd := model.Update(tt.key)
			var got tea.Msg
			if cmd != nil {
				got = cmd()
			}
			if got != tt.want {
				t.Errorf("message = %#v, want %#v", got, tt.want)
			}
		})
	}
}

type item string

func (i item) Init() tea.Cmd                       { return nil }
func (i item) Update(tea.Msg) (tea.Model, tea.Cmd) { return i, nil }
func (i item) View() tea.View                      { return tea.NewView(string(i)) }

type highlightItem struct {
	highlighted bool
}

func (h *highlightItem) Init() tea.Cmd { return nil }

func (h *highlightItem) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if highlighted, ok := msg.(list.HighlightHintEvent); ok {
		h.highlighted = bool(highlighted)
	}
	return h, nil
}

func (*highlightItem) View() tea.View { return tea.NewView("item") }
