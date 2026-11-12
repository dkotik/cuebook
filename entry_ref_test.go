package cuebook

import (
	"errors"
	"testing"
)

func TestEntryRefTargetsAnonymousDuplicateByOccurrence(t *testing.T) {
	t.Parallel()

	const source = `[{Name: "same"}, {Name: "same"}]`
	book, err := New([]byte(source))
	if err != nil {
		t.Fatal(err)
	}
	ref, err := book.GetEntryRef(1)
	if err != nil {
		t.Fatal(err)
	}
	patch, err := book.UpdateFieldValue(ref, "Name", "target")
	if err != nil {
		t.Fatal(err)
	}
	updated, err := patch.ApplyToCueSource([]byte(source))
	if err != nil {
		t.Fatal(err)
	}

	updatedBook, err := New(updated)
	if err != nil {
		t.Fatal(err)
	}
	for index, want := range []string{"same", "target"} {
		entry, err := updatedBook.GetEntryByIndex(index)
		if err != nil {
			t.Fatal(err)
		}
		field, ok := entry.GetFieldByName("Name")
		if !ok {
			t.Fatalf("entry %d has no Name field", index)
		}
		if got := field.String(); got != want {
			t.Errorf("entry %d Name = %q, want %q", index, got, want)
		}
	}
}

func TestEntryRefRejectsDifferentBookSource(t *testing.T) {
	t.Parallel()

	book, err := New([]byte(`[{Name: "Ada"}]`))
	if err != nil {
		t.Fatal(err)
	}
	ref, err := book.GetEntryRef(0)
	if err != nil {
		t.Fatal(err)
	}
	otherBook, err := New([]byte(`[{Name: "Grace"}]`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := otherBook.UpdateFieldValue(ref, "Name", "Lin"); !errors.Is(err, ErrPatchTargetConflict) {
		t.Fatalf("UpdateFieldValue() error = %v, want %v", err, ErrPatchTargetConflict)
	}
}
