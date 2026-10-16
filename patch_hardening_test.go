package cuebook

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
)

func TestPatchApplyRejectsMissingAndMalformedAnchors(t *testing.T) {
	tests := []struct {
		name  string
		patch Patch
	}{
		{
			name: "insert anchor out of bounds",
			patch: insertAfter{
				Preceeding: ByteAnchor{Content: []byte("x"), PreceedingDuplicates: 3},
			},
		},
		{
			name: "delete anchor out of bounds",
			patch: deletePatch{
				Target: ByteAnchor{Content: []byte("x"), PreceedingDuplicates: 3},
			},
		},
		{
			name: "replacement anchor out of bounds",
			patch: replacePatch{
				Target: ByteAnchor{Content: []byte("x"), PreceedingDuplicates: 3},
			},
		},
		{
			name: "negative duplicate count",
			patch: deletePatch{
				Target: ByteAnchor{Content: []byte("x"), PreceedingDuplicates: -1},
			},
		},
		{
			name: "swap anchor missing",
			patch: swapPatch{
				Earlier: ByteAnchor{Content: []byte("missing")},
				Later:   ByteAnchor{Content: []byte("x")},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := test.patch.ApplyToCueSource([]byte("abc"))
			if !errors.Is(err, ErrByteRangeNotFound) {
				t.Fatalf("ApplyToCueSource() error = %v, want %v", err, ErrByteRangeNotFound)
			}
		})
	}
}

func TestByteAnchorMatchFallsBackToLastRemainingDuplicate(t *testing.T) {
	tests := []struct {
		name   string
		source string
		anchor ByteAnchor
		want   ByteRange
	}{
		{
			name:   "all later duplicates disappeared",
			source: "same",
			anchor: ByteAnchor{Content: []byte("same"), PreceedingDuplicates: 2},
			want:   ByteRange{Head: 0, Tail: 4},
		},
		{
			name:   "some later duplicates disappeared",
			source: "same | same",
			anchor: ByteAnchor{Content: []byte("same"), PreceedingDuplicates: 2},
			want:   ByteRange{Head: 7, Tail: 11},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := test.anchor.Match([]byte(test.source))
			if err != nil {
				t.Fatalf("Match() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("Match() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestUpdateRangeRejectsInvalidByteRange(t *testing.T) {
	_, err := UpdateRange([]byte("abc"), ByteRange{Head: 4, Tail: 5}, []byte("x"))
	if !errors.Is(err, ErrByteRangeNotFound) {
		t.Fatalf("UpdateRange() error = %v, want %v", err, ErrByteRangeNotFound)
	}
}

func TestSwapEntriesRejectsOverlappingByteRanges(t *testing.T) {
	source := []byte(`[{Name: "one"}]`)
	document := cuecontext.New().CompileBytes(source)
	if err := document.Err(); err != nil {
		t.Fatal(err)
	}
	entry := document.LookupPath(cuePathIndex(0))
	field := entry.LookupPath(cuePathName("Name"))

	_, err := SwapEntries(source, entry, field)
	if !errors.Is(err, ErrByteRangesOverlap) {
		t.Fatalf("SwapEntries() error = %v, want %v", err, ErrByteRangesOverlap)
	}
}

func TestPatchApplicationDoesNotMutateInput(t *testing.T) {
	tests := []struct {
		name  string
		patch Patch
		want  string
	}{
		{
			name: "insert",
			patch: insertAfter{
				Preceeding: ByteAnchor{Content: []byte("a")},
				Target:     ByteAnchor{Content: []byte("X")},
			},
			want: "aXbc",
		},
		{
			name: "delete",
			patch: deletePatch{
				Target: ByteAnchor{Content: []byte("b")},
			},
			want: "ac",
		},
		{
			name: "replace",
			patch: replacePatch{
				Target:      ByteAnchor{Content: []byte("b")},
				Replacement: ByteAnchor{Content: []byte("X")},
			},
			want: "aXc",
		},
		{
			name: "swap",
			patch: swapPatch{
				Earlier: ByteAnchor{Content: []byte("a")},
				Later:   ByteAnchor{Content: []byte("c")},
			},
			want: "cba",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := []byte("abc")
			original := bytes.Clone(source)
			got, err := test.patch.ApplyToCueSource(source)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != test.want {
				t.Fatalf("result = %q, want %q", got, test.want)
			}
			if !bytes.Equal(source, original) {
				t.Fatalf("input source was modified: %q", source)
			}
		})
	}
}

func TestAppendToEmptySourceCanBeInverted(t *testing.T) {
	value := cuecontext.New().CompileString(`{Name: "Added"}`)
	if err := value.Err(); err != nil {
		t.Fatal(err)
	}

	change, err := AppendToStructList(nil, value)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := change.ApplyToCueSource(nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := New(updated); err != nil {
		t.Fatalf("appended source is invalid: %v\n%s", err, updated)
	}
	restored, err := change.Invert().ApplyToCueSource(updated)
	if err != nil {
		t.Fatal(err)
	}
	if len(restored) != 0 {
		t.Fatalf("inverse result = %q, want empty source", restored)
	}
}

func TestDeleteDifferenceIdentifiesPrecedingContext(t *testing.T) {
	source := []byte("[{Name: \"one\"}, {Name: \"two\"}]")
	document := cuecontext.New().CompileBytes(source)
	if err := document.Err(); err != nil {
		t.Fatal(err)
	}

	value := document.LookupPath(cuePathIndex(1))
	change, err := DeleteFromStructList(source, value)
	if err != nil {
		t.Fatal(err)
	}
	if got := change.Difference().Content; len(got) == 0 || !bytes.Contains(got, []byte(`{Name: "one"}`)) {
		t.Fatalf("Difference() = %q, want context containing the preceding entry", got)
	}
}

func TestResultMatchFindsChangedEntry(t *testing.T) {
	source := []byte(`[{Name: "old"}]`)
	document := cuecontext.New().CompileBytes(source)
	if err := document.Err(); err != nil {
		t.Fatal(err)
	}
	value := document.LookupPath(cuePathIndex(0))
	replacement := []byte(`{Name: "new"}`)
	change, err := ReplaceStructListEntry(source, value, replacement)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := change.ApplyToCueSource(source)
	if err != nil {
		t.Fatal(err)
	}
	book, err := New(updated)
	if err != nil {
		t.Fatal(err)
	}
	result := Result{Document: book, Source: updated, LastChange: change}
	entry, ok := result.Match(replacement)
	if !ok {
		t.Fatal("Match() did not find the changed entry")
	}
	if got, err := entry.Value.LookupPath(cuePathName("Name")).String(); err != nil || got != "new" {
		t.Fatalf("matched Name = %q, err = %v; want %q", got, err, "new")
	}
	withoutChange := Result{Document: book, Source: updated}
	if _, ok := withoutChange.Match(replacement); ok {
		t.Fatal("Match() returned an entry without a last change")
	}
}

func TestResultMatchFindsInsertedEntry(t *testing.T) {
	source := []byte("[]")
	value := cuecontext.New().CompileString(`{Name: "added"}`)
	if err := value.Err(); err != nil {
		t.Fatal(err)
	}
	change, err := AppendToStructList(source, value)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := change.ApplyToCueSource(source)
	if err != nil {
		t.Fatal(err)
	}
	book, err := New(updated)
	if err != nil {
		t.Fatal(err)
	}
	result := Result{Document: book, Source: updated, LastChange: change}
	entry, ok := result.Match(change.Difference().Content)
	if !ok {
		t.Fatal("Match() did not find the inserted entry")
	}
	if got, err := entry.Value.LookupPath(cuePathName("Name")).String(); err != nil || got != "added" {
		t.Fatalf("matched Name = %q, err = %v; want %q", got, err, "added")
	}
}

func TestResultMatchFindsPrecedingEntryAfterDeletion(t *testing.T) {
	source := []byte(`[{Name: "one"}, {Name: "two"}]`)
	document := cuecontext.New().CompileBytes(source)
	if err := document.Err(); err != nil {
		t.Fatal(err)
	}
	change, err := DeleteFromStructList(source, document.LookupPath(cuePathIndex(1)))
	if err != nil {
		t.Fatal(err)
	}
	updated, err := change.ApplyToCueSource(source)
	if err != nil {
		t.Fatal(err)
	}
	book, err := New(updated)
	if err != nil {
		t.Fatal(err)
	}
	result := Result{Document: book, Source: updated, LastChange: change}
	entry, ok := result.Match(change.Difference().Content)
	if !ok {
		t.Fatal("Match() did not find the preceding entry")
	}
	if got, err := entry.Value.LookupPath(cuePathName("Name")).String(); err != nil || got != "one" {
		t.Fatalf("matched Name = %q, err = %v; want %q", got, err, "one")
	}
}

func TestCommitPreservesPermissionsAndCleansTemporaryFile(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "book.cue")
	source := []byte(`[{Name: "old"}]`)
	if err := os.WriteFile(target, source, 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(target, 0o640); err != nil {
		t.Fatal(err)
	}
	document := cuecontext.New().CompileBytes(source)
	if err := document.Err(); err != nil {
		t.Fatal(err)
	}
	change, err := ReplaceStructListEntry(source, document.LookupPath(cuePathIndex(0)), []byte(`{Name: "new"}`))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := Commit(target, directory, change); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o640 {
		t.Fatalf("target permissions = %04o, want 0640", got)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(target) {
		t.Fatalf("directory entries after commit = %v, want only target", entries)
	}
}

func TestCommitCleansTemporaryFileWhenRenameFails(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "book.cue")
	original := []byte("original")
	if err := os.WriteFile(target, original, 0o600); err != nil {
		t.Fatal(err)
	}

	renameErr := errors.New("rename denied")
	err := replaceFileAtomically(target, directory, []byte("replacement"), 0o600, func(string, string) error {
		return renameErr
	})
	if !errors.Is(err, renameErr) {
		t.Fatalf("replaceFileAtomically() error = %v, want %v", err, renameErr)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, original) {
		t.Fatalf("target changed after failed rename: %q", got)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(target) {
		t.Fatalf("directory entries after failed rename = %v, want only target", entries)
	}
}

func TestCommitLeavesOriginalUntouchedWhenSwapDirectoryIsMissing(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "book.cue")
	source := []byte(`[{Name: "old"}]`)
	if err := os.WriteFile(target, source, 0o600); err != nil {
		t.Fatal(err)
	}
	document := cuecontext.New().CompileBytes(source)
	if err := document.Err(); err != nil {
		t.Fatal(err)
	}
	change, err := ReplaceStructListEntry(source, document.LookupPath(cuePathIndex(0)), []byte(`{Name: "new"}`))
	if err != nil {
		t.Fatal(err)
	}

	_, err = Commit(target, filepath.Join(directory, "missing"), change)
	if err == nil {
		t.Fatal("Commit() error = nil, want temporary-file creation error")
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, source) {
		t.Fatalf("target changed after temporary-file creation failed: %q", got)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(target) {
		t.Fatalf("directory entries after failed commit = %v, want only target", entries)
	}
}

func TestCommitLeavesOriginalUntouchedWhenPatchCannotApply(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "book.cue")
	source := []byte(`[{Name: "old"}]`)
	if err := os.WriteFile(target, source, 0o600); err != nil {
		t.Fatal(err)
	}
	change := deletePatch{Target: ByteAnchor{Content: []byte("missing")}}

	if _, err := Commit(target, directory, change); !errors.Is(err, ErrByteRangeNotFound) {
		t.Fatalf("Commit() error = %v, want %v", err, ErrByteRangeNotFound)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, source) {
		t.Fatalf("target changed after patch application failed: %q", got)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(target) {
		t.Fatalf("directory entries after failed commit = %v, want only target", entries)
	}
}

func TestCommitLeavesOriginalUntouchedWhenValidationFails(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "book.cue")
	source := []byte(`[{Name: "old"}]`)
	if err := os.WriteFile(target, source, 0o600); err != nil {
		t.Fatal(err)
	}
	change := replacePatch{
		Target:      ByteAnchor{Content: source},
		Replacement: ByteAnchor{Content: []byte(`"not a list"`)},
	}

	if _, err := Commit(target, directory, change); err == nil {
		t.Fatal("Commit() error = nil, want validation error")
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, source) {
		t.Fatalf("target changed after validation failure: %q", got)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(target) {
		t.Fatalf("directory entries after failed commit = %v, want only target", entries)
	}
}

func cuePathIndex(index int) cue.Path {
	return cue.MakePath(cue.Index(index))
}

func cuePathName(name string) cue.Path {
	return cue.MakePath(cue.Str(name))
}
