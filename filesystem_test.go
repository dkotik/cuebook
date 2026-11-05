package cuebook

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func newTestFileSystem(t *testing.T, files map[string]string) (fileSystem, string) {
	t.Helper()
	root := t.TempDir()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return fileSystem{Directory: os.DirFS(root), Root: root}, root
}

func assertDirectory(t *testing.T, root string, want map[string]string) {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != len(want) {
		names := make([]string, 0, len(entries))
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		t.Fatalf("directory has %v, want %d files (swap files must not be left behind)", names, len(want))
	}
	for name, content := range want {
		got, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if string(got) != content {
			t.Errorf("%s = %q, want %q", name, got, content)
		}
	}
}

func TestFileSystemWriteFile(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		files    map[string]string
		write    string
		content  string
		swap     func(string) (string, fs.File, error)
		wantErr  bool
		want     map[string]string
		wantMode fs.FileMode
	}{
		{
			name:     "replaces existing file and preserves permissions",
			files:    map[string]string{"book.cue": "old"},
			write:    "book.cue",
			content:  "new",
			want:     map[string]string{"book.cue": "new"},
			wantMode: 0o600,
		},
		{
			name:     "creates missing file",
			write:    "new.cue",
			content:  "created",
			want:     map[string]string{"new.cue": "created"},
			wantMode: defaultFileMode,
		},
		{
			name:    "rejects paths outside the root",
			files:   map[string]string{"book.cue": "old"},
			write:   "../escape.cue",
			content: "new",
			wantErr: true,
			want:    map[string]string{"book.cue": "old"},
		},
		{
			name:    "leaves target untouched when swap file cannot be created",
			files:   map[string]string{"book.cue": "old"},
			write:   "book.cue",
			content: "new",
			swap: func(string) (string, fs.File, error) {
				return "", nil, errors.New("disk full")
			},
			wantErr: true,
			want:    map[string]string{"book.cue": "old"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fsys, root := newTestFileSystem(t, tt.files)
			fsys.Swap = tt.swap
			err := fsys.WriteFile(context.Background(), tt.write, []byte(tt.content))
			if (err != nil) != tt.wantErr {
				t.Fatalf("WriteFile error = %v, wantErr %v", err, tt.wantErr)
			}
			assertDirectory(t, root, tt.want)
			if tt.wantMode != 0 {
				info, err := os.Stat(filepath.Join(root, tt.write))
				if err != nil {
					t.Fatal(err)
				}
				if got := info.Mode().Perm(); got != tt.wantMode {
					t.Errorf("mode = %v, want %v", got, tt.wantMode)
				}
			}
			if !tt.wantErr {
				read, err := fsys.ReadFile(context.Background(), tt.write)
				if err != nil || string(read) != tt.content {
					t.Errorf("ReadFile = %q, %v; want %q", read, err, tt.content)
				}
			}
		})
	}
}

func TestFileSystemTransaction(t *testing.T) {
	t.Parallel()
	failure := errors.New("later step failed")
	tests := []struct {
		name     string
		closeErr error
		want     map[string]string
	}{
		{
			name: "keeps changes when closed without an error",
			want: map[string]string{"a.cue": "a2", "b.cue": "b1", "c.cue": "c1"},
		},
		{
			name:     "restores originals and removes created files on error",
			closeErr: failure,
			want:     map[string]string{"a.cue": "a0", "b.cue": "b0"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			fsys, root := newTestFileSystem(t, map[string]string{"a.cue": "a0", "b.cue": "b0"})
			tx := fsys.BeginTransaction(ctx)
			bound, err := fsys.WithTransaction(tx)
			if err != nil {
				t.Fatal(err)
			}
			for _, write := range []struct{ name, content string }{
				{"a.cue", "a1"}, {"b.cue", "b1"}, {"c.cue", "c1"}, {"a.cue", "a2"},
			} {
				if err := bound.WriteFile(ctx, write.name, []byte(write.content)); err != nil {
					t.Fatalf("WriteFile %s: %v", write.name, err)
				}
			}

			err = tt.closeErr
			tx.Close(&err)
			if !errors.Is(err, tt.closeErr) || (tt.closeErr == nil && err != nil) {
				t.Errorf("Close error = %v, want %v", err, tt.closeErr)
			}
			assertDirectory(t, root, tt.want)

			if err := bound.WriteFile(ctx, "a.cue", []byte("late")); !errors.Is(err, ErrTransactionClosed) {
				t.Errorf("write after Close error = %v, want ErrTransactionClosed", err)
			}
			if _, err := fsys.WithTransaction(tx); !errors.Is(err, ErrTransactionClosed) {
				t.Errorf("WithTransaction after Close error = %v, want ErrTransactionClosed", err)
			}
		})
	}
}

type foreignTransaction struct{}

func (foreignTransaction) Close(*error) {}

func TestFileSystemWithTransactionRejectsUnsupportedTransactions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		transaction Transaction
	}{
		{name: "nil", transaction: nil},
		{name: "foreign implementation", transaction: foreignTransaction{}},
		{name: "typed nil", transaction: (*fileTransaction)(nil)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := (fileSystem{}).WithTransaction(tt.transaction); err == nil {
				t.Fatal("WithTransaction error = nil, want error")
			}
		})
	}
}

func TestFileSystemHonorsCanceledContext(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	fsys, root := newTestFileSystem(t, map[string]string{"book.cue": "old"})
	tests := []struct {
		name string
		call func() error
	}{
		{name: "read", call: func() error { _, err := fsys.ReadFile(ctx, "book.cue"); return err }},
		{name: "write", call: func() error { return fsys.WriteFile(ctx, "book.cue", []byte("new")) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(); !errors.Is(err, context.Canceled) {
				t.Errorf("error = %v, want context.Canceled", err)
			}
		})
	}
	assertDirectory(t, root, map[string]string{"book.cue": "old"})
}
