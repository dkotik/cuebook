package htmx

import (
	"context"
	"io/fs"
	"net/http"
	"os"
	"testing/fstest"

	"github.com/dkotik/cuebook"
)

// newTestHandler returns a writable handler. A recordingFileSystem without a
// source reads from source, so tests can share one fixture for both.
func newTestHandler(source fs.FS, files cuebook.FileSystem, opts ...Option) (http.Handler, error) {
	if recorder, ok := files.(*recordingFileSystem); ok && recorder.source == nil {
		recorder.source = source
	}
	return NewWithFileSystem(source, files, opts...)
}

// newDirectoryHandler returns a writable handler for an operating system
// directory, wired the same way as the cuebook serve command.
func newDirectoryHandler(directory string, opts ...Option) (http.Handler, error) {
	files, err := cuebook.NewFileSystem(cuebook.FileSystemConfig{Root: directory})
	if err != nil {
		return nil, err
	}
	return NewWithFileSystem(os.DirFS(directory), files, opts...)
}

// testTransaction satisfies [cuebook.Transaction] for fakes that do not roll
// back; the handler does not use transactions.
type testTransaction struct{}

func (testTransaction) Close(*error) {}

// mapFileSystem reads and writes files in an in-memory [fstest.MapFS], so
// handler reads observe its writes.
type mapFileSystem struct {
	files fstest.MapFS
}

func (m mapFileSystem) BeginTransaction(context.Context) cuebook.Transaction {
	return testTransaction{}
}

func (m mapFileSystem) WithTransaction(cuebook.Transaction) (cuebook.FileSystem, error) {
	return m, nil
}

func (m mapFileSystem) ReadFile(_ context.Context, name string) ([]byte, error) {
	file, ok := m.files[name]
	if !ok {
		return nil, &fs.PathError{Op: "read", Path: name, Err: fs.ErrNotExist}
	}
	return append([]byte(nil), file.Data...), nil
}

func (m mapFileSystem) WriteFile(_ context.Context, name string, content []byte) error {
	data := append([]byte(nil), content...)
	if file, ok := m.files[name]; ok {
		file.Data = data
		return nil
	}
	m.files[name] = &fstest.MapFile{Data: data}
	return nil
}

// recordingFileSystem reads from source and counts writes without applying
// them. Each write returns err.
type recordingFileSystem struct {
	source fs.FS
	calls  int
	err    error
}

func (r *recordingFileSystem) BeginTransaction(context.Context) cuebook.Transaction {
	return testTransaction{}
}

func (r *recordingFileSystem) WithTransaction(cuebook.Transaction) (cuebook.FileSystem, error) {
	return r, nil
}

func (r *recordingFileSystem) ReadFile(_ context.Context, name string) ([]byte, error) {
	return fs.ReadFile(r.source, name)
}

func (r *recordingFileSystem) WriteFile(context.Context, string, []byte) error {
	r.calls++
	return r.err
}

// transferFileSystem records the name of every write in order and fails the
// writes whose 1-based position has an error in failCalls.
type transferFileSystem struct {
	files     fstest.MapFS
	calls     []string
	failCalls map[int]error
}

func (t *transferFileSystem) BeginTransaction(context.Context) cuebook.Transaction {
	return testTransaction{}
}

func (t *transferFileSystem) WithTransaction(cuebook.Transaction) (cuebook.FileSystem, error) {
	return t, nil
}

func (t *transferFileSystem) ReadFile(ctx context.Context, name string) ([]byte, error) {
	return mapFileSystem{files: t.files}.ReadFile(ctx, name)
}

func (t *transferFileSystem) WriteFile(ctx context.Context, name string, content []byte) error {
	t.calls = append(t.calls, name)
	if err := t.failCalls[len(t.calls)]; err != nil {
		return err
	}
	return mapFileSystem{files: t.files}.WriteFile(ctx, name, content)
}

var (
	_ cuebook.FileSystem = mapFileSystem{}
	_ cuebook.FileSystem = (*recordingFileSystem)(nil)
	_ cuebook.FileSystem = (*transferFileSystem)(nil)
)
