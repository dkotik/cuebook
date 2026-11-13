package entry

import (
	"errors"
	"strings"
	"testing"

	"github.com/dkotik/cuebook"
	listForm "github.com/dkotik/cuebook/terminalui/internal/form"
)

func TestEntryContainingUsesUUIDWhenDifferenceAlsoAppearsElsewhere(t *testing.T) {
	const source = `#entry: {ID: string @cuebook(uuid), Name: string, Other: string}
[...#entry] & [{ID: "target", Name: "before", Other: "first"}, {ID: "other", Name: "target", Other: "second"}]`
	book, err := cuebook.New([]byte(source))
	if err != nil {
		t.Fatal(err)
	}
	ref, err := book.GetEntryRef(0)
	if err != nil {
		t.Fatal(err)
	}
	patch, err := book.SwapFields(ref, "Name", "Other")
	if err != nil {
		t.Fatal(err)
	}
	updated, err := patch.ApplyToCueSource([]byte(source))
	if err != nil {
		t.Fatal(err)
	}
	document, err := cuebook.New(updated)
	if err != nil {
		t.Fatal(err)
	}
	wrappedPatch := swapFieldsPatch{Patch: patch}
	result := cuebook.Result{Document: document, Source: updated, LastChange: wrappedPatch}

	got, ok := entryContaining(result, patch.Difference())
	if !ok {
		t.Fatal("entryContaining() did not find the identified entry")
	}
	if got.UUID != "target" {
		t.Errorf("entryContaining() UUID = %q, want %q", got.UUID, "target")
	}
}

func TestFormUpdatePatchRejectsDuplicateUUIDs(t *testing.T) {
	const source = `#entry: {ID: string @cuebook(uuid), Name: string}
[...#entry] & [{ID: "target", Name: "before"}]`
	state, ref, cards := loadEntry(t, source, 0)
	model := openForm(t, state.Document, ref, cards, 0)
	model, _ = model.Update(fieldChangedEvent{Name: "Name", Value: "after", Original: "before"})

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

	duplicateSource := []byte(`#entry: {ID: string @cuebook(uuid), Name: string}
[...#entry] & [{ID: "target", Name: "before"}, {ID: "target", Name: "duplicate"}]`)
	if _, err := patch.ApplyToCueSource(duplicateSource); !errors.Is(err, cuebook.ErrPatchTargetAmbiguous) {
		t.Fatalf("applying form patch to duplicate UUIDs error = %v, want %v", err, cuebook.ErrPatchTargetAmbiguous)
	}
}

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
