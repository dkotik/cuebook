package entry

import (
	"slices"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/terminalui/internal/list"
)

const reorderSchema = `#task: {
	Title: string @cuebook(title)
	Owner: string
	Done: *false | bool
	Notes?: string @cuebook(detail)
	Link?: string @cuebook(detail)
	...
}
`

var (
	ctrlJ    = tea.KeyPressMsg{Code: 'j', Mod: tea.ModCtrl}
	ctrlK    = tea.KeyPressMsg{Code: 'k', Mod: tea.ModCtrl}
	ctrlDown = tea.KeyPressMsg{Code: tea.KeyDown, Mod: tea.ModCtrl}
	ctrlUp   = tea.KeyPressMsg{Code: tea.KeyUp, Mod: tea.ModCtrl}
)

func TestLoadFieldsUsesEntrySourceOrder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source string
		want   []string
	}{
		{
			name:   "fields without a schema",
			source: `[{b: 2, c: 3, a: 1}]`,
			want:   []string{"b", "c", "a"},
		},
		{
			name:   "source order overrides schema order",
			source: reorderSchema + `[...#task] & [{Owner: "Ada", Title: "Plan"}]`,
			want:   []string{"Owner", "Title", "Done", "Notes", "Link"},
		},
		{
			name:   "details follow fields in their own source order",
			source: reorderSchema + `[...#task] & [{Link: "l", Owner: "Ada", Notes: "n", Title: "Plan"}]`,
			want:   []string{"Owner", "Title", "Done", "Link", "Notes"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, cards := loadEntry(t, tt.source, 0)
			if got := orderNames(cards.Order); !slices.Equal(got, tt.want) {
				t.Errorf("field order = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormReordersFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		source       string
		selected     int
		key          tea.KeyPressMsg
		want         string // empty when the key must not change the source
		wantOrder    []string
		wantSelected int
	}{
		{
			name:         "ctrl+j moves the first field down",
			source:       `[{Name: "Ada", Email: "a@b", Role: "dev"}]`,
			key:          ctrlJ,
			want:         `[{Email: "a@b", Name: "Ada", Role: "dev"}]`,
			wantOrder:    []string{"Email", "Name", "Role"},
			wantSelected: 1,
		},
		{
			name:         "ctrl+k moves the last field up",
			source:       `[{Name: "Ada", Email: "a@b", Role: "dev"}]`,
			selected:     2,
			key:          ctrlK,
			want:         `[{Name: "Ada", Role: "dev", Email: "a@b"}]`,
			wantOrder:    []string{"Name", "Role", "Email"},
			wantSelected: 1,
		},
		{
			name:         "ctrl+down moves a field down",
			source:       `[{Name: "Ada", Email: "a@b", Role: "dev"}]`,
			selected:     1,
			key:          ctrlDown,
			want:         `[{Name: "Ada", Role: "dev", Email: "a@b"}]`,
			wantOrder:    []string{"Name", "Role", "Email"},
			wantSelected: 2,
		},
		{
			name:         "ctrl+up moves a field up",
			source:       `[{Name: "Ada", Email: "a@b", Role: "dev"}]`,
			selected:     1,
			key:          ctrlUp,
			want:         `[{Email: "a@b", Name: "Ada", Role: "dev"}]`,
			wantOrder:    []string{"Email", "Name", "Role"},
			wantSelected: 0,
		},
		{
			name:         "only the open entry changes",
			source:       `[{a: 1, b: 2}, {a: 1, b: 2}]`,
			key:          ctrlJ,
			want:         `[{a: 1, b: 2}, {b: 2, a: 1}]`,
			wantOrder:    []string{"b", "a"},
			wantSelected: 1,
		},
		{
			name:         "labels and attributes move with values",
			source:       reorderSchema + `[...#task] & [{Title: "Plan" @cuebook(title), Owner: "Ada"}]`,
			key:          ctrlJ,
			want:         reorderSchema + `[...#task] & [{Owner: "Ada", Title: "Plan" @cuebook(title)}]`,
			wantOrder:    []string{"Owner", "Title", "Done", "Notes", "Link"},
			wantSelected: 1,
		},
		{
			name:         "details reorder among details",
			source:       reorderSchema + `[...#task] & [{Title: "Plan", Owner: "Ada", Notes: "n", Link: "l"}]`,
			selected:     3,
			key:          ctrlJ,
			want:         reorderSchema + `[...#task] & [{Title: "Plan", Owner: "Ada", Link: "l", Notes: "n"}]`,
			wantOrder:    []string{"Title", "Owner", "Done", "Link", "Notes"},
			wantSelected: 4,
		},
		{
			name:     "ctrl+k on the first field does nothing",
			source:   `[{Name: "Ada", Email: "a@b"}]`,
			key:      ctrlK,
			selected: 0,
		},
		{
			name:     "ctrl+j on the last field does not swap with a button",
			source:   `[{Name: "Ada", Email: "a@b"}]`,
			selected: 1,
			key:      ctrlJ,
		},
		{
			name:     "fields that are not written in the entry stay put",
			source:   reorderSchema + `[...#task] & [{Title: "Plan", Owner: "Ada"}]`,
			selected: 1, // Owner; Done comes from the schema default
			key:      ctrlJ,
		},
		{
			name:     "a field and a detail do not trade places",
			source:   "#t: {Title: string, Owner: string, Notes?: string @cuebook(detail)}\n[...#t] & [{Title: \"Plan\", Owner: \"Ada\", Notes: \"n\"}]",
			selected: 1, // Owner; the next card is the Notes detail
			key:      ctrlJ,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			entryIndex := 0
			if tt.name == "only the open entry changes" {
				entryIndex = 1
			}
			state, cards := loadEntry(t, tt.source, entryIndex)
			model := openForm(t, cards, tt.selected)

			patch, ok := pressReorderKey(t, model, tt.key, state)
			if tt.want == "" {
				if ok {
					t.Fatalf("key produced a patch, want none")
				}
				return
			}
			if !ok {
				t.Fatal("key did not produce a patch")
			}
			updated, err := patch.ApplyToCueSource(state.Source)
			if err != nil {
				t.Fatal(err)
			}
			if string(updated) != tt.want {
				t.Fatalf("source =\n%s\nwant\n%s", updated, tt.want)
			}

			book, err := cuebook.New(updated)
			if err != nil {
				t.Fatal(err)
			}
			_, cmd := model.Update(cuebook.Result{Document: book, Source: updated, LastChange: patch})
			if cmd == nil {
				t.Fatal("result did not reload the fields")
			}
			msg := cmd()
			reloaded, ok := msg.(fieldListCards)
			if !ok {
				t.Fatalf("result produced %T, want reloaded fields", msg)
			}
			if got := orderNames(reloaded.Order); !slices.Equal(got, tt.wantOrder) {
				t.Errorf("reloaded order = %q, want %q", got, tt.wantOrder)
			}
			if reloaded.SelectedIndex != tt.wantSelected {
				t.Errorf("selected index = %d, want %d", reloaded.SelectedIndex, tt.wantSelected)
			}
		})
	}
}

func TestFormReorderRequiresSavedChanges(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		changed bool
	}{
		{name: "without unsaved changes"},
		{name: "with unsaved changes", changed: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			state, cards := loadEntry(t, `[{Name: "Ada", Email: "a@b"}]`, 0)
			model := openForm(t, cards, 0)
			if tt.changed {
				model, _ = model.Update(fieldChangedEvent{Name: "Name", Value: "Grace", Original: "Ada"})
			}
			_, ok := pressReorderKey(t, model, ctrlJ, state)
			if ok == tt.changed {
				t.Errorf("patch produced = %t, want %t", ok, !tt.changed)
			}
		})
	}
}

func TestFormDoesNotEmitEntryReorderEvents(t *testing.T) {
	t.Parallel()

	_, cards := loadEntry(t, `[{Name: "Ada", Email: "a@b"}]`, 0)
	model := openForm(t, cards, 0)
	_, cmd := model.Update(ctrlJ)
	if cmd == nil {
		t.Fatal("ctrl+j produced no command")
	}
	switch msg := cmd().(type) {
	case list.SwapOrderEvent:
		t.Fatal("entry view emitted an entry reorder event that the entry list would act on")
	case swapFieldsEvent:
		if msg.CurrentIndex != 0 || msg.DesiredIndex != 1 {
			t.Errorf("swap = %+v, want 0 -> 1", msg)
		}
	default:
		t.Fatalf("ctrl+j produced %T, want a field reorder event", msg)
	}
}

// loadEntry parses source and loads the fields of the entry at index.
func loadEntry(t *testing.T, source string, index int) (cuebook.Result, fieldListCards) {
	t.Helper()
	book, err := cuebook.New([]byte(source))
	if err != nil {
		t.Fatal(err)
	}
	i := 0
	for e, err := range book.EachEntry() {
		if err != nil {
			t.Fatal(err)
		}
		if i == index {
			cards, ok := LoadFields(e)().(fieldListCards)
			if !ok {
				t.Fatal("LoadFields did not return field cards")
			}
			return cuebook.Result{Document: book, Source: []byte(source)}, cards
		}
		i++
	}
	t.Fatalf("entry %d not found", index)
	return cuebook.Result{}, fieldListCards{}
}

func openForm(t *testing.T, cards fieldListCards, selected int) tea.Model {
	t.Helper()
	var model tea.Model = newForm(cards.Entry)
	_ = model.Init()
	model, _ = model.Update(cards)
	model, _ = model.Update(list.ApplySelection(selected)())
	return model
}

// pressReorderKey sends key to the form and follows the resulting commands
// until a patch is produced, as the entry list would when committing it.
func pressReorderKey(t *testing.T, model tea.Model, key tea.KeyPressMsg, state cuebook.Result) (cuebook.Patch, bool) {
	t.Helper()
	_, cmd := model.Update(key)
	if cmd == nil {
		return nil, false
	}
	swap, ok := cmd().(swapFieldsEvent)
	if !ok {
		return nil, false
	}
	_, cmd = model.Update(swap)
	if cmd == nil {
		return nil, false
	}
	update, ok := cmd().(Update)
	if !ok {
		return nil, false
	}
	patch, err := update(state)
	if err != nil {
		t.Fatal(err)
	}
	return patch, true
}

func orderNames(order []orderedField) []string {
	names := make([]string, len(order))
	for i, field := range order {
		names[i] = field.Name
	}
	return names
}
