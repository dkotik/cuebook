package search

import (
	"io/fs"
	"testing"
	"testing/fstest"
	"time"
)

func TestSearchFSIndexesCUEFilesRecursively(t *testing.T) {
	t.Parallel()

	files := fstest.MapFS{
		"root.cue": {
			Data: []byte(`[{Name: "Root record", Text: "rootneedle"}]`),
		},
		"nested/people.cue": {
			Data: []byte(`[{Name: "Nested record", Text: "nestedneedle"}]`),
		},
		"notes.txt": {
			Data: []byte("ignoredword"),
		},
		"invalid.cue": {
			Data: []byte(`[{Name: "Broken record", Text: "brokenmarker"}`),
		},
	}
	indexed, err := NewFS(files)
	if err != nil {
		t.Fatal(err)
	}
	waitForIndex(t, indexed)

	gotSource, err := fs.ReadFile(indexed, "root.cue")
	if err != nil {
		t.Fatalf("read source through search FS: %v", err)
	}
	if string(gotSource) != string(files["root.cue"].Data) {
		t.Errorf("read source = %q, want %q", gotSource, files["root.cue"].Data)
	}

	tests := []struct {
		name      string
		query     string
		wantCount int
		wantPath  string
		wantTitle string
	}{
		{
			name:      "indexes root CUE files",
			query:     "rootneedle",
			wantCount: 1,
			wantPath:  "root.cue",
			wantTitle: "Root record",
		},
		{
			name:      "indexes nested CUE files",
			query:     "nestedneedle",
			wantCount: 1,
			wantPath:  "nested/people.cue",
			wantTitle: "Nested record",
		},
		{
			name:      "ignores non-CUE files",
			query:     "ignoredword",
			wantCount: 0,
		},
		{
			name:      "skips invalid CUE files",
			query:     "brokenmarker",
			wantCount: 0,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			results, err := indexed.Query(test.query)
			if err != nil {
				t.Fatal(err)
			}
			if len(results) != test.wantCount {
				t.Fatalf("result count = %d, want %d", len(results), test.wantCount)
			}
			if test.wantCount == 0 {
				return
			}
			if results[0].Path != test.wantPath {
				t.Errorf("result path = %q, want %q", results[0].Path, test.wantPath)
			}
			if title := results[0].Entry.GetTitle(); title != test.wantTitle {
				t.Errorf("result title = %q, want %q", title, test.wantTitle)
			}
		})
	}
}

func TestSearchFSUpdateFileReplacesIndexedEntries(t *testing.T) {
	t.Parallel()

	const filePath = "people.cue"
	files := fstest.MapFS{
		filePath: {
			Data: []byte(`[{Name: "Old record", Text: "retiredtoken"}]`),
		},
	}
	indexed, err := NewFS(files)
	if err != nil {
		t.Fatal(err)
	}
	waitForIndex(t, indexed)

	files[filePath].Data = []byte(`[{Name: "Updated record", Text: "currenttoken"}]`)
	if err := indexed.UpdateFile(filePath); err != nil {
		t.Fatal(err)
	}

	oldResults, err := indexed.Query("retiredtoken")
	if err != nil {
		t.Fatal(err)
	}
	if len(oldResults) != 0 {
		t.Errorf("stale result count = %d, want 0", len(oldResults))
	}
	newResults, err := indexed.Query("currenttoken")
	if err != nil {
		t.Fatal(err)
	}
	if len(newResults) != 1 {
		t.Fatalf("updated result count = %d, want 1", len(newResults))
	}
	if newResults[0].Path != filePath || newResults[0].Entry.GetTitle() != "Updated record" {
		t.Errorf("updated result = %+v, want path %q and title %q", newResults[0], filePath, "Updated record")
	}
}

func TestSearchFSRemoveFileRemovesOnlyThatFile(t *testing.T) {
	t.Parallel()

	files := fstest.MapFS{
		"first.cue": {
			Data: []byte(`[{Name: "First", Text: "sharedneedle"}]`),
		},
		"second.cue": {
			Data: []byte(`[{Name: "Second", Text: "sharedneedle"}]`),
		},
	}
	indexed, err := NewFS(files)
	if err != nil {
		t.Fatal(err)
	}
	waitForIndex(t, indexed)

	before, err := indexed.Query("sharedneedle")
	if err != nil {
		t.Fatal(err)
	}
	if len(before) != 2 {
		t.Fatalf("result count before removal = %d, want 2", len(before))
	}
	delete(files, "first.cue")
	if err := indexed.RemoveFile("first.cue"); err != nil {
		t.Fatal(err)
	}

	after, err := indexed.Query("sharedneedle")
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != 1 || after[0].Path != "second.cue" {
		t.Fatalf("results after removal = %+v, want only second.cue", after)
	}
}

func waitForIndex(t *testing.T, indexed SearchFS) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !indexed.IndexReady() {
		if time.Now().After(deadline) {
			t.Fatal("search FS did not finish indexing")
		}
		time.Sleep(time.Millisecond)
	}
	if err := indexed.IndexError(); err != nil {
		t.Fatalf("indexing failed: %v", err)
	}
}
