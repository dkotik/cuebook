package entry

import (
	"strings"
	"testing"

	"github.com/dkotik/cuebook"
	listForm "github.com/dkotik/cuebook/terminalui/internal/form"
)

func TestSaveChangesRemovesEmptyOptionalAndUndefinedFields(t *testing.T) {
	t.Parallel()

	const source = `#Entry: {
	Name: string
	Optional?: string
	...
}
[...#Entry] & [{Name: "Ada", Optional: "remove me", Extra: "remove me"}]`
	tests := []struct {
		name            string
		field           string
		original        string
		wantEmptyInFile bool
	}{
		{
			name:     "optional defined field is removed",
			field:    "Optional",
			original: "remove me",
		},
		{
			name:     "field absent from definitions is removed",
			field:    "Extra",
			original: "remove me",
		},
		{
			name:            "required defined field remains empty",
			field:           "Name",
			original:        "Ada",
			wantEmptyInFile: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			state, ref, cards := loadEntry(t, source, 0)
			model := openForm(t, state.Document, ref, cards, 0)
			model, _ = model.Update(fieldChangedEvent{
				Name:     tt.field,
				Value:    "",
				Original: tt.original,
			})

			_, cmd := model.Update(listForm.SaveChangesEvent{})
			if cmd == nil {
				t.Fatal("saving changes produced no command")
			}
			result := cmd()
			update, ok := result.(Update)
			if !ok {
				t.Fatalf("save command returned %T, want Update", result)
			}
			patch, err := update(state)
			if err != nil {
				t.Fatal(err)
			}
			updated, err := patch.ApplyToCueSource(state.Source)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := cuebook.New(updated); err != nil {
				t.Fatalf("updated source is invalid: %v\n%s", err, updated)
			}

			// This source has no other empty string values, so the literal is
			// present only when the changed field was saved as an empty string.
			gotEmptyInFile := strings.Contains(string(updated), `""`)
			if gotEmptyInFile != tt.wantEmptyInFile {
				t.Errorf("empty field present in source = %t, want %t:\n%s", gotEmptyInFile, tt.wantEmptyInFile, updated)
			}
		})
	}
}
