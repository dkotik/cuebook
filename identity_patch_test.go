package cuebook

import (
	"bytes"
	"errors"
	"testing"

	"cuelang.org/go/cue"
)

func TestResolveEntryIdentity(t *testing.T) {
	const schema = `#entry: {ID: string @cuebook(uuid), Name: string}
`
	source := []byte(schema + `[...#entry] & [{ID: "target", Name: "before"}]`)
	book, err := New(source)
	if err != nil {
		t.Fatal(err)
	}
	original, err := book.GetEntryRef(0)
	if err != nil {
		t.Fatal(err)
	}
	identity, ok := original.entryIdentity()
	if !ok {
		t.Fatal("entry has no identity")
	}

	tests := []struct {
		name    string
		source  []byte
		wantErr error
	}{
		{
			name: "unique identity after preceding edit and insertion",
			source: []byte("// unrelated edit\n" + schema +
				`[...#entry] & [{ID: "other", Name: "other"}, {ID: "target", Name: "before"}]`),
		},
		{
			name:    "missing identity",
			source:  []byte(schema + `[...#entry] & [{ID: "other", Name: "other"}]`),
			wantErr: ErrPatchTargetConflict,
		},
		{
			name:    "duplicate identity",
			source:  []byte(schema + `[...#entry] & [{ID: "target", Name: "one"}, {ID: "target", Name: "two"}]`),
			wantErr: ErrPatchTargetAmbiguous,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			entry, err := resolveEntryIdentity(tt.source, identity)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("resolveEntryIdentity() error = %v, want %v", err, tt.wantErr)
			}
			if err == nil {
				if entry.entry.UUID != identity.value {
					t.Errorf("resolved UUID = %q, want %q", entry.entry.UUID, identity.value)
				}
				if got := tt.source[entry.sourceRange.Head:entry.sourceRange.Tail]; !bytes.Equal(got, []byte(`{ID: "target", Name: "before"}`)) {
					t.Errorf("resolved source = %q, want target entry", got)
				}
			}
		})
	}
}

func TestIdentifiedEntryDeletePatch(t *testing.T) {
	const schema = `#entry: {ID: string @cuebook(uuid), Name: string}
`
	tests := []struct {
		name   string
		source string
		index  int
	}{
		{
			name:   "only entry",
			source: schema + `[...#entry] & [{ID: "target", Name: "only"}]`,
			index:  0,
		},
		{
			name:   "first entry",
			source: schema + `[...#entry] & [{ID: "target", Name: "first"}, {ID: "right", Name: "right"}]`,
			index:  0,
		},
		{
			name:   "middle entry",
			source: schema + `[...#entry] & [{ID: "left", Name: "left"}, {ID: "target", Name: "middle"}, {ID: "right", Name: "right"}]`,
			index:  1,
		},
		{
			name:   "last entry",
			source: schema + `[...#entry] & [{ID: "left", Name: "left"}, {ID: "target", Name: "last"}]`,
			index:  1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			source := []byte(tt.source)
			book, err := New(source)
			if err != nil {
				t.Fatal(err)
			}
			entry, err := book.GetEntryRef(tt.index)
			if err != nil {
				t.Fatal(err)
			}
			change, err := book.DeleteFromStructList(entry)
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := change.(identifiedEntryDeletePatch); !ok {
				t.Fatalf("DeleteFromStructList() returned %T, want identifiedEntryDeletePatch", change)
			}

			updated, err := change.ApplyToCueSource(source)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := New(updated); err != nil {
				t.Fatalf("deleted source is invalid: %v\n%s", err, updated)
			}
			retried, err := change.ApplyToCueSource(updated)
			if err != nil {
				t.Fatalf("retry failed: %v", err)
			}
			if !bytes.Equal(retried, updated) {
				t.Fatalf("retry changed source:\n%s\nwant unchanged:\n%s", retried, updated)
			}

			restored, err := change.Invert().ApplyToCueSource(updated)
			if err != nil {
				t.Fatalf("inverse failed: %v", err)
			}
			if !bytes.Equal(restored, source) {
				t.Fatalf("inverse did not restore source:\n%s\nwant:\n%s", restored, source)
			}
			deletedAgain, err := change.Invert().Invert().ApplyToCueSource(restored)
			if err != nil {
				t.Fatalf("inverse of inverse failed: %v", err)
			}
			if !bytes.Equal(deletedAgain, updated) {
				t.Fatalf("inverse of inverse = %q, want %q", deletedAgain, updated)
			}
		})
	}
}

func TestIdentifiedEntryDeletePatchReResolvesShiftedTarget(t *testing.T) {
	const schema = `#entry: {ID: string @cuebook(uuid), Name: string}
`
	source := []byte(schema + `[...#entry] & [{ID: "left", Name: "left"}, {ID: "target", Name: "target"}]`)
	book, err := New(source)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := book.GetEntryRef(1)
	if err != nil {
		t.Fatal(err)
	}
	change, err := book.DeleteFromStructList(entry)
	if err != nil {
		t.Fatal(err)
	}

	shifted := []byte("// unrelated source edit\n" + schema + `[...#entry] & [{ID: "other", Name: "other"}, {ID: "left", Name: "left"}, {ID: "target", Name: "target"}]`)
	updated, err := change.ApplyToCueSource(shifted)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(updated, []byte(`ID: "target"`)) {
		t.Fatalf("target entry was not deleted:\n%s", updated)
	}
	if !bytes.Contains(updated, []byte(`ID: "other"`)) || !bytes.Contains(updated, []byte(`ID: "left"`)) {
		t.Fatalf("unrelated entries were modified:\n%s", updated)
	}
	retried, err := change.ApplyToCueSource(updated)
	if err != nil {
		t.Fatalf("retry on shifted source failed: %v", err)
	}
	if !bytes.Equal(retried, updated) {
		t.Fatalf("retry changed shifted source:\n%s", retried)
	}
	restored, err := change.Invert().ApplyToCueSource(updated)
	if err != nil {
		t.Fatalf("inverse on shifted source failed: %v", err)
	}
	if !bytes.Contains(restored, []byte(`{ID: "other", Name: "other"}, {ID: "left", Name: "left"}, {ID: "target", Name: "target"}`)) {
		t.Fatalf("inverse did not restore target at its current boundary:\n%s", restored)
	}
}

func TestIdentifiedEntryDeletePatchRejectsAmbiguousOrUnexpectedState(t *testing.T) {
	const schema = `#entry: {ID: string @cuebook(uuid), Name: string}
`
	source := []byte(schema + `[...#entry] & [{ID: "left", Name: "left"}, {ID: "target", Name: "target"}, {ID: "right", Name: "right"}]`)
	book, err := New(source)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := book.GetEntryRef(1)
	if err != nil {
		t.Fatal(err)
	}
	change, err := book.DeleteFromStructList(entry)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name   string
		source []byte
		want   error
	}{
		{
			name:   "duplicate target identifier",
			source: []byte(schema + `[...#entry] & [{ID: "left", Name: "left"}, {ID: "target", Name: "target"}, {ID: "target", Name: "duplicate"}, {ID: "right", Name: "right"}]`),
			want:   ErrPatchTargetAmbiguous,
		},
		{
			name:   "target changed independently",
			source: []byte(schema + `[...#entry] & [{ID: "left", Name: "left"}, {ID: "target", Name: "changed"}, {ID: "right", Name: "right"}]`),
			want:   ErrPatchTargetConflict,
		},
		{
			name:   "target missing at unexpected boundary",
			source: []byte(schema + `[...#entry] & [{ID: "left", Name: "left"}, {ID: "other", Name: "other"}, {ID: "right", Name: "right"}]`),
			want:   ErrPatchTargetConflict,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			updated, err := change.ApplyToCueSource(tt.source)
			if !errors.Is(err, tt.want) {
				t.Fatalf("ApplyToCueSource() error = %v, want %v", err, tt.want)
			}
			if updated != nil {
				t.Fatalf("ApplyToCueSource() returned modified bytes on conflict: %q", updated)
			}
		})
	}
}

func TestMergeFieldValuesLatchesToUUIDEntry(t *testing.T) {
	const schema = `#entry: {ID: string @cuebook(uuid), Name: string}
`
	source := []byte(schema + `[...#entry] & [{ID: "left", Name: "left"}, {ID: "target", Name: "before"}]`)
	book, err := New(source)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := book.GetEntryRef(1)
	if err != nil {
		t.Fatal(err)
	}
	change, err := book.MergeFieldValues(entry, map[string]string{"Name": "after"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := change.(identifiedEntryReplacePatch); !ok {
		t.Fatalf("MergeFieldValues() returned %T, want identifiedEntryReplacePatch", change)
	}

	shifted := []byte("// unrelated edit\n" + schema + `[...#entry] & [{ID: "other", Name: "other"}, {ID: "left", Name: "left"}, {ID: "target", Name: "before"}]`)
	updated, err := change.ApplyToCueSource(shifted)
	if err != nil {
		t.Fatal(err)
	}
	updatedBook, err := New(updated)
	if err != nil {
		t.Fatalf("updated source is invalid: %v\n%s", err, updated)
	}
	for index, want := range []string{"other", "left", "after"} {
		current, err := updatedBook.GetEntry(index)
		if err != nil {
			t.Fatal(err)
		}
		got, err := current.Value.LookupPath(cue.ParsePath("Name")).String()
		if err != nil || got != want {
			t.Fatalf("entry %d Name = %q, err = %v; want %q", index, got, err, want)
		}
	}
	retried, err := change.ApplyToCueSource(updated)
	if err != nil {
		t.Fatalf("retry failed: %v", err)
	}
	if !bytes.Equal(retried, updated) {
		t.Fatalf("retry changed source:\n%s", retried)
	}
	restored, err := change.Invert().ApplyToCueSource(updated)
	if err != nil {
		t.Fatalf("inverse failed: %v", err)
	}
	if !bytes.Contains(restored, []byte(`{ID: "target", Name: "before"}`)) {
		t.Fatalf("inverse did not restore target:\n%s", restored)
	}
}

func TestMergeFieldValuesRejectsIdentifierChanges(t *testing.T) {
	const schema = `#entry: {ID?: string @cuebook(uuid), Name: string}
`
	tests := []struct {
		name   string
		source string
		value  string
	}{
		{
			name:   "change existing identifier",
			source: schema + `[...#entry] & [{ID: "target", Name: "before"}]`,
			value:  "different",
		},
		{
			name:   "add missing identifier",
			source: schema + `[...#entry] & [{Name: "before"}]`,
			value:  "new",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			book, err := New([]byte(tt.source))
			if err != nil {
				t.Fatal(err)
			}
			entry, err := book.GetEntryRef(0)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := book.MergeFieldValues(entry, map[string]string{"ID": tt.value}); !errors.Is(err, ErrEntryIdentifierImmutable) {
				t.Fatalf("MergeFieldValues() error = %v, want %v", err, ErrEntryIdentifierImmutable)
			}
		})
	}
}

func TestMergeFieldValuesAndRemoveLatchesToUUIDEntry(t *testing.T) {
	const schema = `#entry: {ID: string @cuebook(uuid), Name: string, Extra?: string}
`
	source := []byte(schema + `[...#entry] & [{ID: "left", Name: "left"}, {ID: "target", Name: "before", Extra: "remove"}]`)
	book, err := New(source)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := book.GetEntryRef(1)
	if err != nil {
		t.Fatal(err)
	}
	change, err := book.MergeFieldValuesAndRemove(entry, map[string]string{"Name": "after"}, []string{"Extra"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := change.(identifiedEntryReplacePatch); !ok {
		t.Fatalf("MergeFieldValuesAndRemove() returned %T, want identifiedEntryReplacePatch", change)
	}

	shifted := []byte("// unrelated edit\n" + schema + `[...#entry] & [{ID: "other", Name: "other"}, {ID: "left", Name: "left"}, {ID: "target", Name: "before", Extra: "remove"}]`)
	updated, err := change.ApplyToCueSource(shifted)
	if err != nil {
		t.Fatal(err)
	}
	updatedBook, err := New(updated)
	if err != nil {
		t.Fatalf("updated source is invalid: %v\n%s", err, updated)
	}
	target, err := updatedBook.GetEntry(2)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := target.Value.LookupPath(cue.ParsePath("Name")).String(); err != nil || got != "after" {
		t.Fatalf("target Name = %q, err = %v; want after", got, err)
	}
	if target.Value.LookupPath(cue.ParsePath("Extra")).Exists() {
		t.Fatalf("Extra was not removed:\n%s", updated)
	}
	if got, err := updatedBook.GetEntry(0); err != nil {
		t.Fatal(err)
	} else if name, err := got.Value.LookupPath(cue.ParsePath("Name")).String(); err != nil || name != "other" {
		t.Fatalf("unrelated entry Name = %q, err = %v; want other", name, err)
	}

	retried, err := change.ApplyToCueSource(updated)
	if err != nil {
		t.Fatalf("retry failed: %v", err)
	}
	if !bytes.Equal(retried, updated) {
		t.Fatalf("retry changed source:\n%s", retried)
	}
	restored, err := change.Invert().ApplyToCueSource(updated)
	if err != nil {
		t.Fatalf("inverse failed: %v", err)
	}
	if !bytes.Contains(restored, []byte(`{ID: "target", Name: "before", Extra: "remove"}`)) {
		t.Fatalf("inverse did not restore original entry:\n%s", restored)
	}
}

func TestMergeFieldValuesAndRemoveRejectsIdentifierChanges(t *testing.T) {
	const schema = `#entry: {ID: string @cuebook(uuid), Name: string}
`
	source := []byte(schema + `[...#entry] & [{ID: "target", Name: "before"}]`)
	book, err := New(source)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := book.GetEntryRef(0)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name     string
		values   map[string]string
		removals []string
	}{
		{name: "change identifier", values: map[string]string{"ID": "different"}},
		{name: "remove identifier", removals: []string{"ID"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := book.MergeFieldValuesAndRemove(entry, tt.values, tt.removals); !errors.Is(err, ErrEntryIdentifierImmutable) {
				t.Fatalf("MergeFieldValuesAndRemove() error = %v, want %v", err, ErrEntryIdentifierImmutable)
			}
		})
	}
}

func TestRemoveFieldValueLatchesToUUIDEntry(t *testing.T) {
	const schema = `#entry: {ID: string @cuebook(uuid), Name: string, Extra?: string}
`
	source := []byte(schema + `[...#entry] & [{ID: "left", Name: "left"}, {ID: "target", Name: "target", Extra: "remove"}]`)
	book, err := New(source)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := book.GetEntryRef(1)
	if err != nil {
		t.Fatal(err)
	}
	change, err := book.RemoveFieldValue(entry, "Extra")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := change.(identifiedEntryReplacePatch); !ok {
		t.Fatalf("RemoveFieldValue() returned %T, want identifiedEntryReplacePatch", change)
	}

	shifted := []byte("// unrelated edit\n" + schema + `[...#entry] & [{ID: "other", Name: "other"}, {ID: "left", Name: "left"}, {ID: "target", Name: "target", Extra: "remove"}]`)
	updated, err := change.ApplyToCueSource(shifted)
	if err != nil {
		t.Fatal(err)
	}
	updatedBook, err := New(updated)
	if err != nil {
		t.Fatalf("updated source is invalid: %v\n%s", err, updated)
	}
	target, err := updatedBook.GetEntry(2)
	if err != nil {
		t.Fatal(err)
	}
	if target.Value.LookupPath(cue.ParsePath("Extra")).Exists() {
		t.Fatalf("Extra was not removed:\n%s", updated)
	}
	retried, err := change.ApplyToCueSource(updated)
	if err != nil {
		t.Fatalf("retry failed: %v", err)
	}
	if !bytes.Equal(retried, updated) {
		t.Fatalf("retry changed source:\n%s", retried)
	}
	restored, err := change.Invert().ApplyToCueSource(updated)
	if err != nil {
		t.Fatalf("inverse failed: %v", err)
	}
	if !bytes.Contains(restored, []byte(`{ID: "target", Name: "target", Extra: "remove"}`)) {
		t.Fatalf("inverse did not restore the field:\n%s", restored)
	}
}

func TestRemoveFieldValueRejectsIdentifierRemoval(t *testing.T) {
	const source = `#entry: {ID: string @cuebook(uuid), Name: string}
[...#entry] & [{ID: "target", Name: "target"}]`
	book, err := New([]byte(source))
	if err != nil {
		t.Fatal(err)
	}
	entry, err := book.GetEntryRef(0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := book.RemoveFieldValue(entry, "ID"); !errors.Is(err, ErrEntryIdentifierImmutable) {
		t.Fatalf("RemoveFieldValue() error = %v, want %v", err, ErrEntryIdentifierImmutable)
	}
}

func TestReplaceStructListEntryPreservesUUID(t *testing.T) {
	const schema = `#entry: {ID: string @cuebook(uuid), Name: string}
`
	source := []byte(schema + `[...#entry] & [{ID: "left", Name: "left"}, {ID: "target", Name: "before"}]`)
	book, err := New(source)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := book.GetEntryRef(1)
	if err != nil {
		t.Fatal(err)
	}
	change, err := book.ReplaceStructListEntry(entry, []byte(`{ID: "target", Name: "after"}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := change.(identifiedEntryReplacePatch); !ok {
		t.Fatalf("ReplaceStructListEntry() returned %T, want identifiedEntryReplacePatch", change)
	}

	shifted := []byte("// unrelated edit\n" + schema + `[...#entry] & [{ID: "other", Name: "other"}, {ID: "left", Name: "left"}, {ID: "target", Name: "before"}]`)
	updated, err := change.ApplyToCueSource(shifted)
	if err != nil {
		t.Fatal(err)
	}
	retried, err := change.ApplyToCueSource(updated)
	if err != nil || !bytes.Equal(retried, updated) {
		t.Fatalf("retry = %q, err = %v; want unchanged result", retried, err)
	}
	restored, err := change.Invert().ApplyToCueSource(updated)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(restored, []byte(`{ID: "target", Name: "before"}`)) {
		t.Fatalf("inverse did not restore original entry:\n%s", restored)
	}
}

func TestReplaceStructListEntryRejectsIdentifierChanges(t *testing.T) {
	const schema = `#entry: {ID: string @cuebook(uuid), Name: string}
`
	source := []byte(schema + `[...#entry] & [{ID: "target", Name: "before"}]`)
	book, err := New(source)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := book.GetEntryRef(0)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name        string
		replacement string
	}{
		{name: "different identifier", replacement: `{ID: "other", Name: "after"}`},
		{name: "missing identifier", replacement: `{Name: "after"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := book.ReplaceStructListEntry(entry, []byte(tt.replacement)); !errors.Is(err, ErrEntryIdentifierImmutable) {
				t.Fatalf("ReplaceStructListEntry() error = %v, want %v", err, ErrEntryIdentifierImmutable)
			}
		})
	}
}

func TestSwapFieldsLatchesToUUIDEntry(t *testing.T) {
	const schema = `#entry: {ID: string @cuebook(uuid), First: string, Second: string}
`
	source := []byte(schema + `[...#entry] & [{ID: "left", First: "left-1", Second: "left-2"}, {ID: "target", First: "before-1", Second: "before-2"}]`)
	book, err := New(source)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := book.GetEntryRef(1)
	if err != nil {
		t.Fatal(err)
	}
	change, err := book.SwapFields(entry, "First", "Second")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := change.(identifiedEntryReplacePatch); !ok {
		t.Fatalf("SwapFields() returned %T, want identifiedEntryReplacePatch", change)
	}

	shifted := []byte("// unrelated edit\n" + schema + `[...#entry] & [{ID: "other", First: "other-1", Second: "other-2"}, {ID: "left", First: "left-1", Second: "left-2"}, {ID: "target", First: "before-1", Second: "before-2"}]`)
	updated, err := change.ApplyToCueSource(shifted)
	if err != nil {
		t.Fatal(err)
	}
	updatedBook, err := New(updated)
	if err != nil {
		t.Fatalf("updated source is invalid: %v\n%s", err, updated)
	}
	target, err := updatedBook.GetEntry(2)
	if err != nil {
		t.Fatal(err)
	}
	for field, want := range map[string]string{"First": "before-1", "Second": "before-2"} {
		got, err := target.Value.LookupPath(cue.MakePath(cue.Str(field))).String()
		if err != nil || got != want {
			t.Fatalf("target %s = %q, err = %v; want %q", field, got, err, want)
		}
	}
	targetRange, err := NewByteRange(target.Value)
	if err != nil {
		t.Fatal(err)
	}
	targetSource := updated[targetRange.Head:targetRange.Tail]
	if bytes.Index(targetSource, []byte(`Second: "before-2"`)) > bytes.Index(targetSource, []byte(`First: "before-1"`)) {
		t.Fatalf("target field declarations were not swapped:\n%s", targetSource)
	}
	retried, err := change.ApplyToCueSource(updated)
	if err != nil || !bytes.Equal(retried, updated) {
		t.Fatalf("retry = %q, err = %v; want unchanged result", retried, err)
	}
	restored, err := change.Invert().ApplyToCueSource(updated)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(restored, []byte(`{ID: "target", First: "before-1", Second: "before-2"}`)) {
		t.Fatalf("inverse did not restore original fields:\n%s", restored)
	}
}

func TestSwapFieldsRejectsIdentifierMutation(t *testing.T) {
	const source = `#entry: {ID: string @cuebook(uuid), Name: string}
[...#entry] & [{ID: "target", Name: "target"}]`
	book, err := New([]byte(source))
	if err != nil {
		t.Fatal(err)
	}
	entry, err := book.GetEntryRef(0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := book.SwapFields(entry, "ID", "Name"); !errors.Is(err, ErrEntryIdentifierImmutable) {
		t.Fatalf("SwapFields() error = %v, want %v", err, ErrEntryIdentifierImmutable)
	}
}

func TestUpdateFieldValueLatchesToUUIDEntry(t *testing.T) {
	const schema = `#entry: {ID: string @cuebook(uuid), Name: string}
`
	source := []byte(schema + `[...#entry] & [{ID: "left", Name: "left"}, {ID: "target", Name: "before"}]`)
	book, err := New(source)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := book.GetEntryRef(1)
	if err != nil {
		t.Fatal(err)
	}
	change, err := book.UpdateFieldValue(entry, "Name", "after")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := change.(identifiedEntryReplacePatch); !ok {
		t.Fatalf("UpdateFieldValue() returned %T, want identifiedEntryReplacePatch", change)
	}

	shifted := []byte("// unrelated edit\n" + schema + `[...#entry] & [{ID: "other", Name: "other"}, {ID: "left", Name: "left"}, {ID: "target", Name: "before"}]`)
	updated, err := change.ApplyToCueSource(shifted)
	if err != nil {
		t.Fatal(err)
	}
	updatedBook, err := New(updated)
	if err != nil {
		t.Fatalf("updated source is invalid: %v\n%s", err, updated)
	}
	target, err := updatedBook.GetEntry(2)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := target.Value.LookupPath(cue.ParsePath("Name")).String(); err != nil || got != "after" {
		t.Fatalf("target Name = %q, err = %v; want after", got, err)
	}
	retried, err := change.ApplyToCueSource(updated)
	if err != nil || !bytes.Equal(retried, updated) {
		t.Fatalf("retry = %q, err = %v; want unchanged result", retried, err)
	}
	restored, err := change.Invert().ApplyToCueSource(updated)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(restored, []byte(`{ID: "target", Name: "before"}`)) {
		t.Fatalf("inverse did not restore target:\n%s", restored)
	}
}

func TestUpdateFieldValueRejectsIdentifierMutation(t *testing.T) {
	const source = `#entry: {ID: string @cuebook(uuid), Name: string}
[...#entry] & [{ID: "target", Name: "target"}]`
	book, err := New([]byte(source))
	if err != nil {
		t.Fatal(err)
	}
	entry, err := book.GetEntryRef(0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := book.UpdateFieldValue(entry, "ID", "different"); !errors.Is(err, ErrEntryIdentifierImmutable) {
		t.Fatalf("UpdateFieldValue() error = %v, want %v", err, ErrEntryIdentifierImmutable)
	}
}

func TestResultMatchFindsNeighborAfterIdentifiedDeletion(t *testing.T) {
	const schema = `#entry: {ID: string @cuebook(uuid), Name: string}
`
	tests := []struct {
		name     string
		source   string
		index    int
		wantUUID string
	}{
		{
			name:     "preceding identified entry",
			source:   schema + `[...#entry] & [{ID: "left", Name: "left"}, {ID: "target", Name: "target"}]`,
			index:    1,
			wantUUID: "left",
		},
		{
			name:     "following identified entry with empty difference content",
			source:   schema + `[...#entry] & [{ID: "target", Name: "target"}, {ID: "right", Name: "right"}]`,
			index:    0,
			wantUUID: "right",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			source := []byte(tt.source)
			book, err := New(source)
			if err != nil {
				t.Fatal(err)
			}
			entry, err := book.GetEntryRef(tt.index)
			if err != nil {
				t.Fatal(err)
			}
			change, err := book.DeleteFromStructList(entry)
			if err != nil {
				t.Fatal(err)
			}
			updated, err := change.ApplyToCueSource(source)
			if err != nil {
				t.Fatal(err)
			}
			updatedBook, err := New(updated)
			if err != nil {
				t.Fatal(err)
			}
			result := Result{Document: updatedBook, Source: updated, LastChange: change}
			matched, ok := result.Match(change.Difference().Content)
			if !ok || matched.UUID != tt.wantUUID {
				t.Fatalf("Match() = (%q, %t), want %q", matched.UUID, ok, tt.wantUUID)
			}
		})
	}
}

func TestMoveEntryLatchesToUUIDAndDestination(t *testing.T) {
	const schema = `#entry: {ID: string @cuebook(uuid), Name: string}
`
	source := []byte(schema + `[...#entry] & [{ID: "a", Name: "A"}, {ID: "b", Name: "B"}, {ID: "c", Name: "C"}]`)
	book, err := New(source)
	if err != nil {
		t.Fatal(err)
	}
	change, err := book.MoveEntry(0, 2)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := change.(identifiedEntryMovePatch); !ok {
		t.Fatalf("MoveEntry() returned %T, want identifiedEntryMovePatch", change)
	}

	shifted := []byte("// unrelated edit\n" + schema + `[...#entry] & [{ID: "a", Name: "A"}, {ID: "b", Name: "B changed"}, {ID: "c", Name: "C changed"}]`)
	updated, err := change.ApplyToCueSource(shifted)
	if err != nil {
		t.Fatal(err)
	}
	updatedBook, err := New(updated)
	if err != nil {
		t.Fatalf("updated source is invalid: %v\n%s", err, updated)
	}
	for index, want := range []string{"b", "c", "a"} {
		entry, err := updatedBook.GetEntry(index)
		if err != nil {
			t.Fatal(err)
		}
		if entry.UUID != want {
			t.Fatalf("entry %d UUID = %q; want %q", index, entry.UUID, want)
		}
	}
	retried, err := change.ApplyToCueSource(updated)
	if err != nil || !bytes.Equal(retried, updated) {
		t.Fatalf("retry = %q, err = %v; want unchanged result", retried, err)
	}
	restored, err := change.Invert().ApplyToCueSource(updated)
	if err != nil {
		t.Fatalf("inverse failed: %v", err)
	}
	restoredBook, err := New(restored)
	if err != nil {
		t.Fatalf("restored source is invalid: %v\n%s", err, restored)
	}
	for index, want := range []string{"a", "b", "c"} {
		entry, err := restoredBook.GetEntry(index)
		if err != nil {
			t.Fatal(err)
		}
		if entry.UUID != want {
			t.Fatalf("restored entry %d UUID = %q; want %q", index, entry.UUID, want)
		}
	}
	b, err := restoredBook.GetEntry(1)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := b.Value.LookupPath(cue.ParsePath("Name")).String(); err != nil || got != "B changed" {
		t.Fatalf("unrelated edit was lost: Name = %q, err = %v", got, err)
	}
}

func TestMoveEntryLatchesToUUIDAtListStart(t *testing.T) {
	const schema = `#entry: {ID: string @cuebook(uuid), Name: string}
`
	source := []byte(schema + `[...#entry] & [{ID: "a", Name: "A"}, {ID: "b", Name: "B"}, {ID: "c", Name: "C"}]`)
	book, err := New(source)
	if err != nil {
		t.Fatal(err)
	}
	change, err := book.MoveEntry(2, 0)
	if err != nil {
		t.Fatal(err)
	}
	shifted := []byte("// unrelated edit\n" + schema + `[...#entry] & [{ID: "a", Name: "A"}, {ID: "b", Name: "B"}, {ID: "c", Name: "C"}]`)
	updated, err := change.ApplyToCueSource(shifted)
	if err != nil {
		t.Fatal(err)
	}
	updatedBook, err := New(updated)
	if err != nil {
		t.Fatal(err)
	}
	for index, want := range []string{"c", "a", "b"} {
		entry, err := updatedBook.GetEntry(index)
		if err != nil {
			t.Fatal(err)
		}
		if entry.UUID != want {
			t.Fatalf("entry %d UUID = %q; want %q", index, entry.UUID, want)
		}
	}
	restored, err := change.Invert().ApplyToCueSource(updated)
	if err != nil {
		t.Fatal(err)
	}
	restoredBook, err := New(restored)
	if err != nil {
		t.Fatal(err)
	}
	for index, want := range []string{"a", "b", "c"} {
		entry, err := restoredBook.GetEntry(index)
		if err != nil {
			t.Fatal(err)
		}
		if entry.UUID != want {
			t.Fatalf("restored entry %d UUID = %q; want %q", index, entry.UUID, want)
		}
	}
}

func TestMoveEntryRejectsAmbiguousOrConflictingIdentity(t *testing.T) {
	const schema = `#entry: {ID: string @cuebook(uuid), Name: string}
`
	source := []byte(schema + `[...#entry] & [{ID: "a", Name: "A"}, {ID: "b", Name: "B"}, {ID: "c", Name: "C"}]`)
	book, err := New(source)
	if err != nil {
		t.Fatal(err)
	}
	change, err := book.MoveEntry(0, 2)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		source []byte
		want   error
	}{
		{
			name:   "duplicate target ID",
			source: []byte(schema + `[...#entry] & [{ID: "a", Name: "A"}, {ID: "b", Name: "B"}, {ID: "a", Name: "duplicate"}]`),
			want:   ErrPatchTargetAmbiguous,
		},
		{
			name:   "missing target ID",
			source: []byte(schema + `[...#entry] & [{ID: "b", Name: "B"}, {ID: "c", Name: "C"}]`),
			want:   ErrPatchTargetConflict,
		},
		{
			name:   "target content changed",
			source: []byte(schema + `[...#entry] & [{ID: "a", Name: "changed"}, {ID: "b", Name: "B"}, {ID: "c", Name: "C"}]`),
			want:   ErrPatchTargetConflict,
		},
		{
			name:   "unexpected order",
			source: []byte(schema + `[...#entry] & [{ID: "a", Name: "A"}, {ID: "c", Name: "C"}, {ID: "b", Name: "B"}]`),
			want:   ErrPatchTargetConflict,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			updated, err := change.ApplyToCueSource(tt.source)
			if !errors.Is(err, tt.want) {
				t.Fatalf("ApplyToCueSource() error = %v, want %v", err, tt.want)
			}
			if updated != nil {
				t.Fatalf("ApplyToCueSource() returned modified bytes on conflict: %q", updated)
			}
		})
	}
}

func TestSwapEntriesLatchesToUUIDIdentities(t *testing.T) {
	const schema = `#entry: {ID: string @cuebook(uuid), Name: string}
`
	source := []byte(schema + `[...#entry] & [{ID: "a", Name: "A"}, {ID: "b", Name: "B"}, {ID: "c", Name: "C"}]`)
	book, err := New(source)
	if err != nil {
		t.Fatal(err)
	}
	a, err := book.GetEntryRef(0)
	if err != nil {
		t.Fatal(err)
	}
	c, err := book.GetEntryRef(2)
	if err != nil {
		t.Fatal(err)
	}
	change, err := book.SwapEntries(a, c)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := change.(identifiedEntrySwapPatch); !ok {
		t.Fatalf("SwapEntries() returned %T, want identifiedEntrySwapPatch", change)
	}

	shifted := []byte("// unrelated edit\n" + schema + `[...#entry] & [{ID: "a", Name: "A"}, {ID: "b", Name: "B changed"}, {ID: "c", Name: "C"}]`)
	updated, err := change.ApplyToCueSource(shifted)
	if err != nil {
		t.Fatal(err)
	}
	updatedBook, err := New(updated)
	if err != nil {
		t.Fatalf("updated source is invalid: %v\n%s", err, updated)
	}
	for index, want := range []string{"c", "b", "a"} {
		entry, err := updatedBook.GetEntry(index)
		if err != nil {
			t.Fatal(err)
		}
		if entry.UUID != want {
			t.Fatalf("entry %d UUID = %q; want %q", index, entry.UUID, want)
		}
	}
	middle, err := updatedBook.GetEntry(1)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := middle.Value.LookupPath(cue.ParsePath("Name")).String(); err != nil || got != "B changed" {
		t.Fatalf("unrelated edit was lost: Name = %q, err = %v", got, err)
	}
	retried, err := change.ApplyToCueSource(updated)
	if err != nil || !bytes.Equal(retried, updated) {
		t.Fatalf("retry = %q, err = %v; want unchanged result", retried, err)
	}
	restored, err := change.Invert().ApplyToCueSource(updated)
	if err != nil {
		t.Fatal(err)
	}
	restoredBook, err := New(restored)
	if err != nil {
		t.Fatalf("restored source is invalid: %v\n%s", err, restored)
	}
	for index, want := range []string{"a", "b", "c"} {
		entry, err := restoredBook.GetEntry(index)
		if err != nil {
			t.Fatal(err)
		}
		if entry.UUID != want {
			t.Fatalf("restored entry %d UUID = %q; want %q", index, entry.UUID, want)
		}
	}
}

func TestSwapEntriesFallsBackWithoutTwoIdentifiers(t *testing.T) {
	const source = `#entry: {ID?: string @cuebook(uuid), Name: string}
[...#entry] & [{ID: "a", Name: "A"}, {Name: "B"}]`
	book, err := New([]byte(source))
	if err != nil {
		t.Fatal(err)
	}
	a, err := book.GetEntryRef(0)
	if err != nil {
		t.Fatal(err)
	}
	b, err := book.GetEntryRef(1)
	if err != nil {
		t.Fatal(err)
	}
	change, err := book.SwapEntries(a, b)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := change.(replacePatch); !ok {
		t.Fatalf("SwapEntries() returned %T, want byte-anchor replacePatch", change)
	}
}

func TestSwapEntriesRejectsAmbiguousOrConflictingIdentity(t *testing.T) {
	const schema = `#entry: {ID: string @cuebook(uuid), Name: string}
`
	source := []byte(schema + `[...#entry] & [{ID: "a", Name: "A"}, {ID: "b", Name: "B"}, {ID: "c", Name: "C"}]`)
	book, err := New(source)
	if err != nil {
		t.Fatal(err)
	}
	a, err := book.GetEntryRef(0)
	if err != nil {
		t.Fatal(err)
	}
	c, err := book.GetEntryRef(2)
	if err != nil {
		t.Fatal(err)
	}
	change, err := book.SwapEntries(a, c)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		source []byte
		want   error
	}{
		{
			name:   "duplicate identifier",
			source: []byte(schema + `[...#entry] & [{ID: "a", Name: "A"}, {ID: "b", Name: "B"}, {ID: "a", Name: "duplicate"}]`),
			want:   ErrPatchTargetAmbiguous,
		},
		{
			name:   "missing identifier",
			source: []byte(schema + `[...#entry] & [{ID: "a", Name: "A"}, {ID: "b", Name: "B"}]`),
			want:   ErrPatchTargetConflict,
		},
		{
			name:   "target changed independently",
			source: []byte(schema + `[...#entry] & [{ID: "a", Name: "changed"}, {ID: "b", Name: "B"}, {ID: "c", Name: "C"}]`),
			want:   ErrPatchTargetConflict,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			updated, err := change.ApplyToCueSource(tt.source)
			if !errors.Is(err, tt.want) {
				t.Fatalf("ApplyToCueSource() error = %v, want %v", err, tt.want)
			}
			if updated != nil {
				t.Fatalf("ApplyToCueSource() returned modified bytes on conflict: %q", updated)
			}
		})
	}
}

func TestIdentifiedEntryReplacePatch(t *testing.T) {
	const schema = `#entry: {ID: string @cuebook(uuid), Name: string}
`
	originalSource := []byte(schema + `[...#entry] & [{ID: "target", Name: "before"}]`)
	book, err := New(originalSource)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := book.GetEntryRef(0)
	if err != nil {
		t.Fatal(err)
	}
	patch, err := book.ReplaceStructListEntry(entry, []byte(`{ID: "target", Name: "after"}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := patch.(identifiedEntryReplacePatch); !ok {
		t.Fatalf("entryReplacementPatch() returned %T, want identifiedEntryReplacePatch", patch)
	}

	freshSource := []byte("// unrelated edit\n" + schema +
		`[...#entry] & [{ID: "other", Name: "other"}, {ID: "target", Name: "before"}]`)
	updated, err := patch.ApplyToCueSource(freshSource)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(updated, []byte(`{ID: "target", Name: "after"}`)) {
		t.Fatalf("updated source does not contain replacement entry:\n%s", updated)
	}

	retried, err := patch.ApplyToCueSource(updated)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(retried, updated) {
		t.Fatalf("retry changed source:\n%s\nwant unchanged:\n%s", retried, updated)
	}

	restored, err := patch.Invert().ApplyToCueSource(updated)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(restored, freshSource) {
		t.Fatalf("inverse did not restore source:\n%s\nwant:\n%s", restored, freshSource)
	}

	updatedBook, err := New(updated)
	if err != nil {
		t.Fatal(err)
	}
	matched, ok := (Result{Document: updatedBook, Source: updated, LastChange: patch}).Match(patch.Difference().Content)
	if !ok || matched.UUID != "target" {
		t.Fatalf("Result.Match() = (%q, %t), want target entry", matched.UUID, ok)
	}

	conflictingSource := []byte(schema + `[...#entry] & [{ID: "target", Name: "changed independently"}]`)
	if _, err := patch.ApplyToCueSource(conflictingSource); !errors.Is(err, ErrPatchTargetConflict) {
		t.Fatalf("ApplyToCueSource() error = %v, want %v", err, ErrPatchTargetConflict)
	}
}
