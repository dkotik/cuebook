package form_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/dkotik/cuebook/terminalui/internal/form"
	"github.com/dkotik/cuebook/terminalui/internal/label"
	"github.com/dkotik/cuebook/terminalui/internal/list"
)

func TestFormLabelLayout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		configuration form.Configuration
		width         int
		want          label.LayoutChangeEvent
	}{
		{
			name:  "default narrow window stacks labels",
			width: 100, // 100 * 0.15 = 15 < 20
			want:  label.LayoutChangeEvent{Width: 15},
		},
		{
			name:  "default wide window places labels beside fields",
			width: 200, // 200 * 0.15 = 30 >= 20
			want:  label.LayoutChangeEvent{Width: 30, IsHorizontal: true},
		},
		{
			name:          "negative values use defaults",
			configuration: form.Configuration{LabelMinimumHorizontalWidth: -1, LabelWidthRatioToWindowWidth: -1},
			width:         200,
			want:          label.LayoutChangeEvent{Width: 30, IsHorizontal: true},
		},
		{
			name:          "custom minimum width",
			configuration: form.Configuration{LabelMinimumHorizontalWidth: 40},
			width:         200,
			want:          label.LayoutChangeEvent{Width: 30},
		},
		{
			name:          "custom width ratio",
			configuration: form.Configuration{LabelWidthRatioToWindowWidth: 0.25},
			width:         100,
			want:          label.LayoutChangeEvent{Width: 25, IsHorizontal: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			recorder := &layoutRecorder{}
			model := form.New(tt.configuration)
			model, _ = model.Init()
			model, _ = model.Update(list.SetItems(recorder)())
			model, _ = model.Update(tea.WindowSizeMsg{Width: tt.width, Height: 20})
			if len(recorder.layouts) != 1 {
				t.Fatalf("received %d layout events, want 1", len(recorder.layouts))
			}
			if got := recorder.layouts[0]; got != tt.want {
				t.Errorf("layout = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestFormStyle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		style lipgloss.Style
	}{
		{name: "default style"},
		{name: "custom style", style: lipgloss.NewStyle().Foreground(lipgloss.Green).AlignVertical(lipgloss.Top)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			render := func(model tea.Model) string {
				model, _ = model.Init()
				model, _ = model.Update(list.SetItems(text("field"))())
				model, _ = model.Update(tea.WindowSizeMsg{Width: 40, Height: 5})
				return model.View()
			}
			got := render(form.New(form.Configuration{Style: tt.style}))
			want := render(list.New(list.Configuration{Style: tt.style}))
			if got != want {
				t.Errorf("View() =\n%q\nwant list view\n%q", got, want)
			}
		})
	}
}

func TestFormSaveShortcut(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		key      tea.KeyPressMsg
		wantSave bool
	}{
		{name: "ctrl+s saves", key: tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl}, wantSave: true},
		{name: "plain s does not save", key: tea.KeyPressMsg{Code: 's'}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, cmd := form.New(form.Configuration{}).Update(tt.key)
			saved := false
			if cmd != nil {
				_, saved = cmd().(form.SaveChangesEvent)
			}
			if saved != tt.wantSave {
				t.Errorf("saved = %t, want %t", saved, tt.wantSave)
			}
		})
	}
}

type layoutRecorder struct {
	layouts []label.LayoutChangeEvent
}

func (r *layoutRecorder) Init() (tea.Model, tea.Cmd) { return r, nil }

func (r *layoutRecorder) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if layout, ok := msg.(label.LayoutChangeEvent); ok {
		r.layouts = append(r.layouts, layout)
	}
	return r, nil
}

func (*layoutRecorder) View() string { return "" }

type text string

func (t text) Init() (tea.Model, tea.Cmd)          { return t, nil }
func (t text) Update(tea.Msg) (tea.Model, tea.Cmd) { return t, nil }
func (t text) View() string                        { return string(t) }
