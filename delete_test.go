package cuebook

import (
	"bytes"
	"errors"
	"slices"
	"testing"

	"cuelang.org/go/cue"
)

func TestDeleteStructFromList(t *testing.T) {
	source := []byte(`[
		{ one: "ok" },
		{	two: "ok" },
]`)
	document, err := New(source)
	if err != nil {
		t.Fatal(err)
	}

	for entry, err := range document.EachEntryRef() {
		if err != nil {
			t.Fatal(err)
		}
		patch, err := document.DeleteFromStructList(entry)
		if err != nil {
			t.Fatal(err)
		}
		patch = Validated(patch)
		t.Run("reverse", ensureInversible(source, patch))
	}

	// TODO: multiple delete and insert cycles leave whitespace artifacts
	last, err := document.GetEntryRef(1)
	if err != nil {
		t.Fatal(err)
	}
	book := document
	changed := slices.Clone(source)
	for range 5 {
		inverse, err := book.AppendToStructList(last.Entry().Value)
		if err != nil {
			t.Fatal(err)
		}
		changed, err = inverse.ApplyToCueSource(changed)
		if err != nil {
			t.Fatal(err)
		}

		updatedBook, err := New(changed)
		if err != nil {
			t.Fatal(err)
		}
		last, err = updatedBook.GetEntryRef(2)
		if err != nil {
			t.Fatal(err)
		}
		patch, err := updatedBook.DeleteFromStructList(last)
		if err != nil {
			t.Fatal(err)
		}
		changed, err = patch.ApplyToCueSource(changed)
		if err != nil {
			t.Fatal(err)
		}
	}

	// TODO: check if there is any extra whitespace
	// t.Fatal(string(changed))
}

func TestRemoveFieldValue(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		declaration string
		field       string
		input       string
		unchanged   bool
	}{
		{name: "true", declaration: "Enabled?: bool", field: "Enabled", input: ", Enabled: true"},
		{name: "false", declaration: "Enabled?: bool", field: "Enabled", input: ", Enabled: false"},
		{name: "optional default", declaration: "Enabled?: *true | bool", field: "Enabled", input: ", Enabled: false"},
		{name: "quoted label", declaration: "\"is enabled\"?: bool", field: "is enabled", input: ", \"is enabled\": true"},
		{name: "already absent", declaration: "Enabled?: bool", field: "Enabled", unchanged: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			item := `{Name: "Existing", Other: false, Secret: "stored-secret"` + tt.input + `}`
			source := []byte("#entry: {Name: string, Other: bool, Secret: string @cuebook(argon2id), Missing?: *true | bool, " + tt.declaration + "}\n[...#entry] & [" + item + ", " + item + "]")
			book, err := New(source)
			if err != nil {
				t.Fatal(err)
			}
			entry, err := book.GetEntryRef(1)
			if err != nil {
				t.Fatal(err)
			}
			change, err := book.RemoveFieldValue(entry, tt.field)
			if err != nil {
				t.Fatal(err)
			}
			updated, err := change.ApplyToCueSource(source)
			if err != nil {
				t.Fatal(err)
			}
			if tt.unchanged && !bytes.Equal(updated, source) {
				t.Fatal("absent field changed the source")
			}
			book, err = New(updated)
			if err != nil {
				t.Fatalf("invalid result: %v\n%s", err, updated)
			}
			value, err := book.GetValue(1)
			if err != nil {
				t.Fatal(err)
			}
			if value.LookupPath(cue.MakePath(cue.Str(tt.field))).Exists() {
				t.Fatalf("removed field still exists: %s", updated)
			}
			if value.LookupPath(cue.ParsePath("Missing")).Exists() {
				t.Fatal("unrelated optional default was materialized")
			}
			if got, err := value.LookupPath(cue.ParsePath("Other")).Bool(); err != nil || got {
				t.Fatalf("Other = %v, err = %v; want false", got, err)
			}
			if got, err := value.LookupPath(cue.ParsePath("Secret")).String(); err != nil || got != "stored-secret" {
				t.Fatalf("Secret = %q, err = %v", got, err)
			}
			if !tt.unchanged {
				first, err := book.GetValue(0)
				if err != nil {
					t.Fatal(err)
				}
				if !first.LookupPath(cue.MakePath(cue.Str(tt.field))).Exists() {
					t.Fatal("wrong duplicate entry was modified")
				}
			}
			restored, err := change.Invert().ApplyToCueSource(updated)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(restored, source) {
				t.Fatalf("inverse did not restore original source:\n%s", restored)
			}
		})
	}
}

func TestRemoveFieldValueRejectsNonStruct(t *testing.T) {
	t.Parallel()
	book, err := New([]byte(`["not a struct"]`))
	if err != nil {
		t.Fatal(err)
	}
	ref, refErr := book.GetEntryRef(0)
	if !errors.Is(refErr, ErrEntryValueNotStruct) {
		t.Fatalf("GetEntryRef() error = %v, want %v", refErr, ErrEntryValueNotStruct)
	}
	_, err = book.RemoveFieldValue(ref, "Enabled")
	if !errors.Is(err, ErrEntryNotStruct) {
		t.Fatalf("error = %v, want %v", err, ErrEntryNotStruct)
	}
}
