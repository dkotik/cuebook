package internal

import (
	"bytes"
	"context"
	"errors"

	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/terminalui/internal/entry"
)

func TestArchiveEntry(t *testing.T) {
	tests := []struct {
		name           string
		source         string
		initialArchive string
		wantSource     []string
		wantArchive    []string
		wantError      string
	}{
		{
			name: "creates daily archive",
			source: `#entry: {Name: string @cuebook(title)}
[...#entry] & [{Name: "Move me"}, {Name: "Keep me"}]
`,
			wantSource:  []string{"Keep me"},
			wantArchive: []string{"Move me"},
		},
		{
			name: "appends to existing archive",
			source: `#entry: {Name: string @cuebook(title)}
[...#entry] & [{Name: "Move me"}, {Name: "Keep me"}]
`,
			initialArchive: `#entry: {Name: string @cuebook(title)}
[...#entry] & [{Name: "Already archived"}]
`,
			wantSource:  []string{"Keep me"},
			wantArchive: []string{"Already archived", "Move me"},
		},
		{
			name: "keeps archive if source constraints reject removal",
			source: `#entry: {Name: string @cuebook(title)}
[...#entry] & [{Name: "Move me"}, {Name: "Keep me"}] & [_, _, ...]
`,
			wantSource:  []string{"Move me", "Keep me"},
			wantArchive: []string{"Move me"},
			wantError:   "unable to remove entry from source",
		},
	}

	date := time.Date(2026, time.November, 6, 12, 0, 0, 0, time.UTC)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			source := []byte(test.source)
			sourceName := "source.cue"
			if err := os.WriteFile(filepath.Join(root, sourceName), source, 0o600); err != nil {
				t.Fatal(err)
			}
			archiveName := cuebook.ArchiveFileName(date)
			archivePath := filepath.Join(root, filepath.FromSlash(archiveName))
			if test.initialArchive != "" {
				if err := os.MkdirAll(filepath.Dir(archivePath), 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(archivePath, []byte(test.initialArchive), 0o600); err != nil {
					t.Fatal(err)
				}
			}

			files, err := cuebook.NewFileSystem(cuebook.FileSystemConfig{Root: root})
			if err != nil {
				t.Fatal(err)
			}
			book, err := cuebook.New(source)
			if err != nil {
				t.Fatal(err)
			}
			entry, err := book.GetEntry(0)
			if err != nil {
				t.Fatal(err)
			}
			_, err = archiveEntry(context.Background(), files, sourceName, entry, date)
			if test.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantError) {
					t.Fatalf("archive error = %v, want containing %q", err, test.wantError)
				}
			} else if err != nil {
				t.Fatal(err)
			}

			updatedSource, err := os.ReadFile(filepath.Join(root, sourceName))
			if err != nil {
				t.Fatal(err)
			}
			updatedArchive, err := os.ReadFile(archivePath)
			if err != nil {
				t.Fatal(err)
			}
			if test.wantError != "" && !bytes.Equal(updatedSource, source) {
				t.Fatalf("source changed after failed source update:\n%s", updatedSource)
			}
			if got := archiveTitles(t, updatedSource); !slices.Equal(got, test.wantSource) {
				t.Errorf("source titles = %q, want %q", got, test.wantSource)
			}
			if got := archiveTitles(t, updatedArchive); !slices.Equal(got, test.wantArchive) {
				t.Errorf("archive titles = %q, want %q", got, test.wantArchive)
			}
		})
	}
}

func TestEntryListArchiveUsesSourceDirectory(t *testing.T) {
	t.Parallel()

	source := []byte(`#entry: {Name: string @cuebook(title)}
[...#entry] & [{Name: "Move me"}, {Name: "Keep me"}]
`)
	root := t.TempDir()
	sourceDirectory := filepath.Join(root, "nested")
	if err := os.Mkdir(sourceDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	sourcePath := filepath.Join(sourceDirectory, "source.cue")
	if err := os.WriteFile(sourcePath, source, 0o600); err != nil {
		t.Fatal(err)
	}
	book, err := cuebook.New(source)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := book.GetEntry(0)
	if err != nil {
		t.Fatal(err)
	}

	model, ok := NewEntryList(sourcePath).(EntryList)
	if !ok {
		t.Fatalf("NewEntryList returned %T, want EntryList", model)
	}
	model.book = cuebook.Result{Document: book, Source: source}
	_, cmd := model.Update(entry.ArchiveEntryEvent{Entry: selected})
	if cmd == nil {
		t.Fatal("archive action did not schedule a command")
	}
	switch msg := cmd().(type) {
	case error:
		t.Fatal(msg)
	case entry.ArchiveEntryCompletedEvent:
	default:
		t.Fatalf("archive command returned %T, want ArchiveEntryCompletedEvent", msg)
	}

	archiveDirectory := filepath.Join(sourceDirectory, ".archive")
	archives, err := os.ReadDir(archiveDirectory)
	if err != nil {
		t.Fatal(err)
	}
	if len(archives) != 1 {
		t.Fatalf("archive files beside source = %d, want 1", len(archives))
	}
	archivedSource, err := os.ReadFile(filepath.Join(archiveDirectory, archives[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := archiveTitles(t, archivedSource), []string{"Move me"}; !slices.Equal(got, want) {
		t.Fatalf("archived titles = %q, want %q", got, want)
	}
	updatedSource, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := archiveTitles(t, updatedSource), []string{"Keep me"}; !slices.Equal(got, want) {
		t.Fatalf("source titles = %q, want %q", got, want)
	}
	if _, err := os.Stat(filepath.Join(root, ".archive")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unexpected archive folder beside parent directory: err = %v", err)
	}
}

func TestArchiveEntryDoesNotRemoveSourceWhenArchiveWriteFails(t *testing.T) {
	t.Parallel()

	const sourceName = "source.cue"
	date := time.Date(2026, time.November, 6, 12, 0, 0, 0, time.UTC)
	archiveName := cuebook.ArchiveFileName(date)
	source := []byte(`#entry: {Name: string @cuebook(title)}
[...#entry] & [{Name: "Move me"}, {Name: "Keep me"}]
`)
	initialArchive := []byte(`[{Name: "Already archived"}]`)

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, sourceName), source, 0o600); err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(root, filepath.FromSlash(archiveName))
	if err := os.MkdirAll(filepath.Dir(archivePath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(archivePath, initialArchive, 0o600); err != nil {
		t.Fatal(err)
	}

	files, err := cuebook.NewFileSystem(cuebook.FileSystemConfig{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	book, err := cuebook.New(source)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := book.GetEntry(0)
	if err != nil {
		t.Fatal(err)
	}
	failingFiles := &writeFailureFileSystem{FileSystem: files, failName: archiveName}
	if _, err := archiveEntry(context.Background(), failingFiles, sourceName, selected, date); err == nil {
		t.Fatal("archiveEntry succeeded despite a failed archive write")
	}

	updatedSource, err := os.ReadFile(filepath.Join(root, sourceName))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(updatedSource, source) {
		t.Fatalf("source changed after archive write failure:\n%s", updatedSource)
	}
	updatedArchive, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(updatedArchive, initialArchive) {
		t.Fatalf("archive changed after failed write:\n%s", updatedArchive)
	}
	if !slices.Equal(failingFiles.writes, []string{archiveName}) {
		t.Fatalf("write calls = %q, want only the archive write", failingFiles.writes)
	}
}

func TestArchiveEntryKeepsArchivedCopyWhenSourceWriteFails(t *testing.T) {
	t.Parallel()

	const sourceName = "source.cue"
	date := time.Date(2026, time.November, 6, 12, 0, 0, 0, time.UTC)
	archiveName := cuebook.ArchiveFileName(date)
	source := []byte(`#entry: {Name: string @cuebook(title)}
[...#entry] & [{Name: "Move me"}, {Name: "Keep me"}]
`)
	initialArchive := []byte(`[{Name: "Already archived"}]`)

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, sourceName), source, 0o600); err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(root, filepath.FromSlash(archiveName))
	if err := os.MkdirAll(filepath.Dir(archivePath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(archivePath, initialArchive, 0o600); err != nil {
		t.Fatal(err)
	}

	files, err := cuebook.NewFileSystem(cuebook.FileSystemConfig{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	book, err := cuebook.New(source)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := book.GetEntry(0)
	if err != nil {
		t.Fatal(err)
	}
	failingFiles := &writeFailureFileSystem{FileSystem: files, failName: sourceName}
	if _, err := archiveEntry(context.Background(), failingFiles, sourceName, selected, date); err == nil {
		t.Fatal("archiveEntry succeeded despite a failed source write")
	}

	updatedSource, err := os.ReadFile(filepath.Join(root, sourceName))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(updatedSource, source) {
		t.Fatalf("source changed after failed source write:\n%s", updatedSource)
	}
	updatedArchive, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := archiveTitles(t, updatedArchive), []string{"Already archived", "Move me"}; !slices.Equal(got, want) {
		t.Fatalf("archive titles = %q, want %q", got, want)
	}
	if !slices.Equal(failingFiles.writes, []string{archiveName, sourceName}) {
		t.Fatalf("write order = %q, want archive then source", failingFiles.writes)
	}
}

type writeFailureFileSystem struct {
	cuebook.FileSystem
	failName string
	writes   []string
}

func (f *writeFailureFileSystem) WriteFile(ctx context.Context, name string, content []byte) error {
	f.writes = append(f.writes, name)
	if name == f.failName {
		return errors.New("simulated file write failure")
	}
	return f.FileSystem.WriteFile(ctx, name, content)
}

func TestArchiveCompletionRefreshesEntryList(t *testing.T) {
	t.Parallel()

	after := []byte(`[{Name: "Keep me"}]`)
	book, err := cuebook.New(after)
	if err != nil {
		t.Fatal(err)
	}
	result := cuebook.Result{Document: book, Source: after}

	model, cmd := (EntryList{}).Update(entry.ArchiveEntryCompletedEvent{
		Title:        "Move me",
		Result:       result,
		ReturnToList: true,
	})
	updated, ok := model.(EntryList)
	if !ok {
		t.Fatalf("updated model = %T, want EntryList", model)
	}
	if !bytes.Equal(updated.book.Source, after) {
		t.Errorf("entry list source = %q, want %q", updated.book.Source, after)
	}
	if cmd == nil {
		t.Fatal("archive completion did not schedule a list refresh")
	}

}

func archiveTitles(t *testing.T, source []byte) []string {
	t.Helper()
	book, err := cuebook.New(source)
	if err != nil {
		t.Fatal(err)
	}
	var titles []string
	for entry, err := range book.EachEntry() {
		if err != nil {
			t.Fatal(err)
		}
		titles = append(titles, entry.Title)
	}
	return titles
}
