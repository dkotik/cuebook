package cuebook

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/cuecontext"
)

// commitInDirectory commits change to the file named name inside directory.
// A nil swap uses the default swap file strategy.
func commitInDirectory(t *testing.T, directory, name string, change Patch, swap func(string) (string, fs.File, error)) (Result, error) {
	t.Helper()
	files, err := NewFileSystem(FileSystemConfig{Root: directory, Swap: swap})
	if err != nil {
		t.Fatal(err)
	}
	return Commit(context.Background(), files, name, change)
}

type writeCountingFileSystem struct {
	FileSystem
	writes int
}

func (f *writeCountingFileSystem) WriteFile(ctx context.Context, name string, source []byte) error {
	f.writes++
	return f.FileSystem.WriteFile(ctx, name, source)
}

func TestCommitSkipsWriteWhenPatchWasAlreadyApplied(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "book.cue")
	source := []byte(`[{Name: "old"}]`)
	if err := os.WriteFile(target, source, 0o600); err != nil {
		t.Fatal(err)
	}
	base, err := NewFileSystem(FileSystemConfig{Root: directory})
	if err != nil {
		t.Fatal(err)
	}
	files := &writeCountingFileSystem{FileSystem: base}
	book, err := New(source)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := book.GetEntryRef(0)
	if err != nil {
		t.Fatal(err)
	}
	patch, err := book.ReplaceStructListEntry(entry, []byte(`{Name: "new"}`))
	if err != nil {
		t.Fatal(err)
	}

	first, err := Commit(context.Background(), files, "book.cue", patch)
	if err != nil {
		t.Fatal(err)
	}
	if first.LastChange == nil {
		t.Fatal("first Commit() LastChange = nil, want the applied patch")
	}
	second, err := Commit(context.Background(), files, "book.cue", patch)
	if err != nil {
		t.Fatal(err)
	}
	if second.LastChange != nil {
		t.Fatal("retry Commit() LastChange is non-nil, want nil for a no-op")
	}
	if files.writes != 1 {
		t.Fatalf("WriteFile calls = %d, want 1", files.writes)
	}
}

func TestPatchApplication(t *testing.T) {
	source := []byte(`[
{ one: "ok" },
{	another: "ok" },
{	another: "ok" },
{	two: "ok" },
{	two: "ok" },
{	two: "ok" }
]`)
	target := filepath.Join(t.TempDir(), "test.cue")
	err := os.WriteFile(target, source, 0700)
	if err != nil {
		t.Fatal(err)
	}
	book, err := New(source)
	if err != nil {
		t.Fatal(err)
	}
	p, err := book.AppendToStructList(cuecontext.New().BuildExpr(
		ast.NewStruct(
			&ast.Field{
				Label: ast.NewString("Name"),
				Value: ast.NewString("Someone"),
			},
			&ast.Field{
				Label: ast.NewString("Email"),
				Value: ast.NewString("someEmail@somehost.net"),
			},
		),
	))
	if err != nil {
		t.Fatal(err)
	}
	result, err := commitInDirectory(t, filepath.Dir(target), filepath.Base(target), p, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.LastChange == nil {
		t.Fatal("Commit() result has no last change")
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(result.Source) {
		t.Fatalf("committed source = %q, result source = %q", got, result.Source)
	}
	if count, err := result.Document.Len(); err != nil || count != 7 {
		t.Fatalf("committed document length = %d, err = %v; want 7", count, err)
	}
}

func TestResultBottomChangeIndex(t *testing.T) {
	tests := []struct {
		name          string
		currentSource string
		olderSource   string
		want          int
	}{
		{
			name:          "unchanged document",
			currentSource: `[{Name: "one"}, {Name: "two"}]`,
			olderSource:   `[{Name: "one"}, {Name: "two"}]`,
			want:          -1,
		},
		{
			name:          "first entry changed",
			currentSource: `[{Name: "updated"}, {Name: "two"}]`,
			olderSource:   `[{Name: "one"}, {Name: "two"}]`,
			want:          0,
		},
		{
			name:          "later entry changed",
			currentSource: `[{Name: "one"}, {Name: "updated"}]`,
			olderSource:   `[{Name: "one"}, {Name: "two"}]`,
			want:          1,
		},
		{
			name:          "new entry appended",
			currentSource: `[{Name: "one"}, {Name: "two"}]`,
			olderSource:   `[{Name: "one"}]`,
			want:          1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			currentSource := []byte(test.currentSource)
			current, err := New(currentSource)
			if err != nil {
				t.Fatal(err)
			}
			olderSource := []byte(test.olderSource)
			older, err := New(olderSource)
			if err != nil {
				t.Fatal(err)
			}
			got, err := (Result{Document: current, Source: currentSource}).BottomChangeIndex(
				Result{Document: older, Source: olderSource},
			)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("BottomChangeIndex() = %d, want %d", got, test.want)
			}
		})
	}
}

func TestResultBottomChangeIndexReturnsListError(t *testing.T) {
	current, err := New([]byte("[]"))
	if err != nil {
		t.Fatal(err)
	}
	invalidList := cuecontext.New().CompileString(`"not a list"`)

	_, err = (Result{Document: current}).BottomChangeIndex(Result{
		Document: Book{Value: invalidList},
	})
	if err == nil {
		t.Fatal("BottomChangeIndex() error = nil, want an error for an invalid list")
	}
}

func ensureInversible(
	source []byte,
	patch Patch,
) func(*testing.T) {
	return func(t *testing.T) {
		intermediate, err := patch.ApplyToCueSource(source)
		if err != nil {
			t.Log("original:", string(source))
			t.Log("intermediate:", string(intermediate))
			t.Fatal("unable to apply patch:", err)
		}
		reversed, err := patch.Invert().ApplyToCueSource(intermediate)
		if err != nil {
			t.Fatal("unable to apply reverse:", err)
		}
		if string(source) != string(reversed) {
			t.Log("original:", string(source))
			t.Log("intermediate:", string(intermediate))
			t.Log("reversed:", string(reversed))
			t.Fatal("reversed patch does not match original")
		}
	}
}

func TestPatchApplyRejectsMissingAndAmbiguousAnchors(t *testing.T) {
	tests := []struct {
		name   string
		source string
		patch  Patch
		want   error
	}{
		{
			name:   "target does not match either state",
			source: "abc",
			patch: replacePatch{
				Target:      ByteAnchor{Content: []byte("missing")},
				Replacement: ByteAnchor{Content: []byte("replacement")},
			},
			want: ErrPatchTargetConflict,
		},
		{
			name:   "target duplicate count is out of bounds",
			source: "abc",
			patch: replacePatch{
				Target:      ByteAnchor{Content: []byte("x"), PreceedingDuplicates: 3},
				Replacement: ByteAnchor{Content: []byte("y")},
			},
			want: ErrPatchTargetConflict,
		},
		{
			name:   "negative duplicate count",
			source: "abc",
			patch: replacePatch{
				Target:      ByteAnchor{Content: []byte("x"), PreceedingDuplicates: -1},
				Replacement: ByteAnchor{Content: []byte("y"), PreceedingDuplicates: -1},
			},
			want: ErrPatchTargetConflict,
		},
		{
			name:   "ambiguous target",
			source: "aba",
			patch: replacePatch{
				Target:      ByteAnchor{Content: []byte("a")},
				Replacement: ByteAnchor{Content: []byte("x")},
			},
			want: ErrPatchTargetAmbiguous,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := test.patch.ApplyToCueSource([]byte(test.source))
			if !errors.Is(err, test.want) {
				t.Fatalf("ApplyToCueSource() error = %v, want %v", err, test.want)
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
	target := (ByteRange{Head: 4, Tail: 5}).Anchor([]byte("abc"))
	if _, err := UpdateRange(target, []byte("x")); !errors.Is(err, ErrByteRangeNotFound) {
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

	entryRange, err := NewByteRange(entry)
	if err != nil {
		t.Fatal(err)
	}
	fieldRange, err := NewByteRange(field)
	if err != nil {
		t.Fatal(err)
	}
	_, err = swapByteRanges(source, entryRange, fieldRange)
	if !errors.Is(err, ErrByteRangesOverlap) {
		t.Fatalf("SwapEntries() error = %v, want %v", err, ErrByteRangesOverlap)
	}
}

func TestPatchApplicationDoesNotMutateInput(t *testing.T) {
	tests := []struct {
		name      string
		makePatch func([]byte) (Patch, error)
		want      string
	}{
		{
			name: "insert",
			makePatch: func(source []byte) (Patch, error) {
				return UpdateRange((ByteRange{Head: 1, Tail: 1}).Anchor(source), []byte("X"))
			},
			want: "aXbc",
		},
		{
			name: "delete",
			makePatch: func(source []byte) (Patch, error) {
				return UpdateRange((ByteRange{Head: 1, Tail: 2}).Anchor(source), nil)
			},
			want: "ac",
		},
		{
			name: "replace",
			makePatch: func(source []byte) (Patch, error) {
				return UpdateRange((ByteRange{Head: 1, Tail: 2}).Anchor(source), []byte("X"))
			},
			want: "aXc",
		},
		{
			name: "swap",
			makePatch: func(source []byte) (Patch, error) {
				return swapByteRanges(source, ByteRange{Head: 0, Tail: 1}, ByteRange{Head: 2, Tail: 3})
			},
			want: "cba",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := []byte("abc")
			original := bytes.Clone(source)
			patch, err := test.makePatch(source)
			if err != nil {
				t.Fatal(err)
			}
			got, err := patch.ApplyToCueSource(source)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != test.want {
				t.Fatalf("result = %q, want %q", got, test.want)
			}
			retried, err := patch.ApplyToCueSource(got)
			if err != nil {
				t.Fatalf("retry failed: %v", err)
			}
			if !bytes.Equal(retried, got) {
				t.Fatalf("retry result = %q, want unchanged %q", retried, got)
			}
			if !bytes.Equal(source, original) {
				t.Fatalf("input source was modified: %q", source)
			}
		})
	}
}

func TestGeneratedPatchesAreIdempotentAndInvertBeforeApplication(t *testing.T) {
	entryPatch := func(source []byte, index int, create func(Book, EntryRef) (Patch, error)) (Patch, error) {
		book, err := New(source)
		if err != nil {
			return nil, err
		}
		entry, err := book.GetEntryRef(index)
		if err != nil {
			return nil, err
		}
		return create(book, entry)
	}
	bookPatch := func(source []byte, create func(Book) (Patch, error)) (Patch, error) {
		book, err := New(source)
		if err != nil {
			return nil, err
		}
		return create(book)
	}
	tests := []struct {
		name      string
		source    string
		makePatch func([]byte) (Patch, error)
	}{
		{
			name:   "replace entry",
			source: `[{Name: "old"}]`,
			makePatch: func(source []byte) (Patch, error) {
				return entryPatch(source, 0, func(book Book, entry EntryRef) (Patch, error) {
					return book.ReplaceStructListEntry(entry, []byte(`{Name: "new"}`))
				})
			},
		},
		{
			name:   "append entry",
			source: `[{Name: "old"}]`,
			makePatch: func(source []byte) (Patch, error) {
				return bookPatch(source, func(book Book) (Patch, error) {
					value := cuecontext.New().CompileString(`{Name: "new"}`)
					return book.AppendToStructList(value)
				})
			},
		},
		{
			name:   "delete entry",
			source: `[{Name: "one"}, {Name: "two"}]`,
			makePatch: func(source []byte) (Patch, error) {
				return entryPatch(source, 1, func(book Book, entry EntryRef) (Patch, error) {
					return book.DeleteFromStructList(entry)
				})
			},
		},
		{
			name:   "update field",
			source: `[{Name: "old"}]`,
			makePatch: func(source []byte) (Patch, error) {
				return entryPatch(source, 0, func(book Book, entry EntryRef) (Patch, error) {
					return book.UpdateFieldValue(entry, "Name", "new")
				})
			},
		},
		{
			name:   "merge fields",
			source: `[{Name: "old"}]`,
			makePatch: func(source []byte) (Patch, error) {
				return entryPatch(source, 0, func(book Book, entry EntryRef) (Patch, error) {
					return book.MergeFieldValues(entry, map[string]string{"Name": "new"})
				})
			},
		},
		{
			name:   "remove field",
			source: `[{Name: "old", Extra: "remove"}]`,
			makePatch: func(source []byte) (Patch, error) {
				return entryPatch(source, 0, func(book Book, entry EntryRef) (Patch, error) {
					return book.RemoveFieldValue(entry, "Extra")
				})
			},
		},
		{
			name:   "add field",
			source: `[{Name: "old"}]`,
			makePatch: func(source []byte) (Patch, error) {
				return entryPatch(source, 0, func(book Book, entry EntryRef) (Patch, error) {
					return book.AddField(entry, "Extra", FieldKindString, "added")
				})
			},
		},
		{
			name:   "swap entries",
			source: `[{Name: "one"}, {Name: "two"}]`,
			makePatch: func(source []byte) (Patch, error) {
				return bookPatch(source, func(book Book) (Patch, error) {
					a, err := book.GetEntryRef(0)
					if err != nil {
						return nil, err
					}
					b, err := book.GetEntryRef(1)
					if err != nil {
						return nil, err
					}
					return book.SwapEntries(a, b)
				})
			},
		},
		{
			name:   "swap fields",
			source: `[{a: 1, b: 2}]`,
			makePatch: func(source []byte) (Patch, error) {
				return entryPatch(source, 0, func(book Book, entry EntryRef) (Patch, error) {
					return book.SwapFields(entry, "a", "b")
				})
			},
		},
		{
			name:   "move entry",
			source: `[{Name: "one"}, {Name: "two"}, {Name: "three"}]`,
			makePatch: func(source []byte) (Patch, error) {
				return bookPatch(source, func(book Book) (Patch, error) {
					return book.MoveEntry(0, 2)
				})
			},
		},
		{
			name:   "range replacement",
			source: "abc",
			makePatch: func(source []byte) (Patch, error) {
				return UpdateRange((ByteRange{Head: 1, Tail: 2}).Anchor(source), []byte("X"))
			},
		},
		{
			name:   "empty source replacement",
			source: "",
			makePatch: func(source []byte) (Patch, error) {
				return UpdateRange((ByteRange{}).Anchor(source), []byte("x"))
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := []byte(test.source)
			patch, err := test.makePatch(source)
			if err != nil {
				t.Fatal(err)
			}
			inverse := patch.Invert()
			updated, err := patch.ApplyToCueSource(source)
			if err != nil {
				t.Fatal(err)
			}
			retried, err := patch.ApplyToCueSource(updated)
			if err != nil {
				t.Fatalf("retry failed: %v", err)
			}
			if !bytes.Equal(retried, updated) {
				t.Fatalf("retry changed source:\n%s\nwant unchanged:\n%s", retried, updated)
			}
			restored, err := inverse.ApplyToCueSource(updated)
			if err != nil {
				t.Fatalf("inverse created before application failed: %v", err)
			}
			if !bytes.Equal(restored, source) {
				t.Fatalf("inverse result:\n%s\nwant original:\n%s", restored, source)
			}
		})
	}
}

func TestAppendToEmptyListCanBeInvertedAndRetried(t *testing.T) {
	source := []byte("[]")
	value := cuecontext.New().CompileString(`{Name: "Added"}`)
	if err := value.Err(); err != nil {
		t.Fatal(err)
	}

	book, err := New(source)
	if err != nil {
		t.Fatal(err)
	}
	change, err := book.AppendToStructList(value)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := change.ApplyToCueSource(source)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := New(updated); err != nil {
		t.Fatalf("appended source is invalid: %v\n%s", err, updated)
	}
	retried, err := change.ApplyToCueSource(updated)
	if err != nil {
		t.Fatalf("retry failed: %v", err)
	}
	if !bytes.Equal(retried, updated) {
		t.Fatalf("retry changed source: %q, want %q", retried, updated)
	}
	restored, err := change.Invert().ApplyToCueSource(updated)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(restored, source) {
		t.Fatalf("inverse result = %q, want %q", restored, source)
	}
}

func TestDeleteDifferenceIdentifiesPrecedingContext(t *testing.T) {
	source := []byte("[{Name: \"one\"}, {Name: \"two\"}]")
	document, err := New(source)
	if err != nil {
		t.Fatal(err)
	}

	entry, err := document.GetEntryRef(1)
	if err != nil {
		t.Fatal(err)
	}
	change, err := document.DeleteFromStructList(entry)
	if err != nil {
		t.Fatal(err)
	}
	if got := change.Difference().Content; len(got) == 0 || !bytes.Contains(got, []byte(`{Name: "one"}`)) {
		t.Fatalf("Difference() = %q, want context containing the preceding entry", got)
	}
}

func TestResultMatchFindsChangedEntry(t *testing.T) {
	source := []byte(`[{Name: "old"}]`)
	document, err := New(source)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := document.GetEntryRef(0)
	if err != nil {
		t.Fatal(err)
	}
	replacement := []byte(`{Name: "new"}`)
	change, err := document.ReplaceStructListEntry(entry, replacement)
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
	matched, ok := result.Match(replacement)
	if !ok {
		t.Fatal("Match() did not find the changed entry")
	}
	if got, err := matched.Value.LookupPath(cuePathName("Name")).String(); err != nil || got != "new" {
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
	book, err := New(source)
	if err != nil {
		t.Fatal(err)
	}
	change, err := book.AppendToStructList(value)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := change.ApplyToCueSource(source)
	if err != nil {
		t.Fatal(err)
	}
	book, err = New(updated)
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
	document, err := New(source)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := document.GetEntryRef(1)
	if err != nil {
		t.Fatal(err)
	}
	change, err := document.DeleteFromStructList(entry)
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
	matched, ok := result.Match(change.Difference().Content)
	if !ok {
		t.Fatal("Match() did not find the preceding entry")
	}
	if got, err := matched.Value.LookupPath(cuePathName("Name")).String(); err != nil || got != "one" {
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
	document, err := New(source)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := document.GetEntryRef(0)
	if err != nil {
		t.Fatal(err)
	}
	change, err := document.ReplaceStructListEntry(entry, []byte(`{Name: "new"}`))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := commitInDirectory(t, directory, "book.cue", change, nil); err != nil {
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
	if runtime.GOOS == "windows" {
		t.Skip("directory permissions do not block renames on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	parent := t.TempDir()
	directory := filepath.Join(parent, "book")
	swapDirectory := filepath.Join(parent, "swap")
	for _, path := range []string{directory, swapDirectory} {
		if err := os.Mkdir(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	target := filepath.Join(directory, "book.cue")
	source := []byte(`[{Name: "old"}]`)
	if err := os.WriteFile(target, source, 0o600); err != nil {
		t.Fatal(err)
	}
	document, err := New(source)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := document.GetEntryRef(0)
	if err != nil {
		t.Fatal(err)
	}
	change, err := document.ReplaceStructListEntry(entry, []byte(`{Name: "new"}`))
	if err != nil {
		t.Fatal(err)
	}

	// The swap file is created in a writable sibling directory, so it can be
	// written and cleaned up, but renaming it into the read-only target
	// directory fails.
	stageInSwapDirectory := func(targetPath string) (string, fs.File, error) {
		file, err := os.CreateTemp(swapDirectory, "."+filepath.Base(targetPath)+"-*.tmp")
		if err != nil {
			return "", nil, err
		}
		return file.Name(), file, nil
	}
	if err := os.Chmod(directory, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(directory, 0o755) })

	_, err = commitInDirectory(t, directory, "book.cue", change, stageInSwapDirectory)
	if !errors.Is(err, fs.ErrPermission) {
		t.Fatalf("Commit() error = %v, want %v", err, fs.ErrPermission)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, source) {
		t.Fatalf("target changed after failed rename: %q", got)
	}
	for _, check := range []struct {
		path string
		want int
	}{
		{path: directory, want: 1},
		{path: swapDirectory, want: 0},
	} {
		entries, err := os.ReadDir(check.path)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != check.want {
			t.Fatalf("%s entries after failed rename = %v, want %d (swap file must be removed)", check.path, entries, check.want)
		}
	}
}

func TestCommitLeavesOriginalUntouchedWhenSwapDirectoryIsMissing(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "book.cue")
	source := []byte(`[{Name: "old"}]`)
	if err := os.WriteFile(target, source, 0o600); err != nil {
		t.Fatal(err)
	}
	document, err := New(source)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := document.GetEntryRef(0)
	if err != nil {
		t.Fatal(err)
	}
	change, err := document.ReplaceStructListEntry(entry, []byte(`{Name: "new"}`))
	if err != nil {
		t.Fatal(err)
	}

	missingSwapDirectory := func(targetPath string) (string, fs.File, error) {
		file, err := os.CreateTemp(filepath.Join(directory, "missing"), "."+filepath.Base(targetPath)+"-*.tmp")
		if err != nil {
			return "", nil, err
		}
		return file.Name(), file, nil
	}
	_, err = commitInDirectory(t, directory, "book.cue", change, missingSwapDirectory)
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
	change := replacePatch{
		Target:      ByteAnchor{Content: []byte("missing")},
		Replacement: ByteAnchor{Content: []byte("replacement")},
	}

	if _, err := commitInDirectory(t, directory, "book.cue", change, nil); !errors.Is(err, ErrPatchTargetConflict) {
		t.Fatalf("Commit() error = %v, want %v", err, ErrPatchTargetConflict)
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

	if _, err := commitInDirectory(t, directory, "book.cue", change, nil); err == nil {
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
