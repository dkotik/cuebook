package cuebook

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// ErrTransactionClosed reports a write through a [Transaction] that was
// already closed.
var ErrTransactionClosed = errors.New("transaction is closed")

const (
	// defaultFileMode is applied to files that did not exist before a write.
	defaultFileMode fs.FileMode = 0o644
	// defaultDirectoryMode is applied to missing parent directories created by
	// a write.
	defaultDirectoryMode fs.FileMode = 0o755
)

// Transaction is a set of changes to be applied atomically to a [FileSystem].
type Transaction interface {
	// Close reverses all the changes made during
	// the [FileSystem] lifetime if the pointer is not nil.
	Close(*error)
}

type FileSystem interface {
	BeginTransaction(context.Context) Transaction
	WithTransaction(Transaction) (FileSystem, error)
	ReadFile(context.Context, string) ([]byte, error)
	WriteFile(context.Context, string, []byte) error
}

type fileSystem struct {
	// Directory serves reads. It must expose the same files as Root.
	Directory fs.FS
	// Root is the operating system directory that writes are renamed into.
	Root string
	// Swap creates a temporary file for the given target path and returns its
	// path. It must live on the same filesystem as the target so that
	// [os.Rename] is atomic. Nil uses [defaultSwapFileStrategy].
	Swap func(string) (string, fs.File, error)

	transaction *fileTransaction
}

var _ FileSystem = fileSystem{}

// FileSystemConfig configures [NewFileSystem].
type FileSystemConfig struct {
	// Root is the operating system directory that holds the files. Required.
	Root string

	// Swap creates a writable temporary file for a target path and returns
	// the temporary file's path. The file must be on the same filesystem as
	// the target so that [os.Rename] replaces the target atomically.
	// Defaults to creating the temporary file next to the target.
	Swap func(targetPath string) (string, fs.File, error)
}

// NewFileSystem returns a [FileSystem] that reads and writes files under
// config.Root. Files are replaced atomically: each write goes to a temporary
// file that is then moved over the original with [os.Rename].
func NewFileSystem(config FileSystemConfig) (FileSystem, error) {
	if config.Root == "" {
		return nil, errors.New("file system root is empty")
	}
	root, err := filepath.Abs(config.Root)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve file system root: %w", err)
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("unable to open file system root: %w", err)
	}
	if !info.IsDir() {
		return nil, &fs.PathError{Op: "open", Path: root, Err: errors.New("not a directory")}
	}
	if config.Swap == nil {
		config.Swap = defaultSwapFileStrategy
	}
	return fileSystem{
		Directory: os.DirFS(root),
		Root:      root,
		Swap:      config.Swap,
	}, nil
}

// BeginTransaction starts recording original file contents so that
// [Transaction.Close] can restore them. Bind it with [fileSystem.WithTransaction].
func (f fileSystem) BeginTransaction(context.Context) Transaction {
	return &fileTransaction{seen: make(map[string]struct{})}
}

// WithTransaction returns a copy of the file system whose writes are recorded
// in t.
func (f fileSystem) WithTransaction(t Transaction) (FileSystem, error) {
	tx, ok := t.(*fileTransaction)
	if !ok || tx == nil {
		return nil, fmt.Errorf("unsupported transaction type %T", t)
	}
	tx.mu.Lock()
	defer tx.mu.Unlock()
	if tx.closed {
		return nil, ErrTransactionClosed
	}
	f.transaction = tx
	return f, nil
}

// ReadFile reads the named file from Directory.
func (f fileSystem) ReadFile(ctx context.Context, name string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return fs.ReadFile(f.Directory, name)
}

// WriteFile atomically replaces the named file with content: content is
// written to a swap file, which is then moved over the target with
// [os.Rename]. Existing permissions are preserved. Missing parent directories
// are created, and paths through symbolic links are refused so that writes
// cannot escape Root.
func (f fileSystem) WriteFile(ctx context.Context, name string, content []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !fs.ValidPath(name) || name == "." {
		return &fs.PathError{Op: "write", Path: name, Err: fs.ErrInvalid}
	}
	targetPath, err := f.resolveTarget(name)
	if err != nil {
		return err
	}

	// The lock is held before the snapshot so that a concurrent write in the
	// same transaction cannot be recorded as the original content.
	if tx := f.transaction; tx != nil {
		tx.mu.Lock()
		defer tx.mu.Unlock()
		if tx.closed {
			return ErrTransactionClosed
		}
	}
	original, mode, existed, err := f.snapshot(name)
	if err != nil {
		return err
	}
	if tx := f.transaction; tx != nil {
		if _, ok := tx.seen[targetPath]; !ok {
			tx.seen[targetPath] = struct{}{}
			tx.undo = append(tx.undo, func() error {
				if !existed {
					if err := os.Remove(targetPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
						return err
					}
					return nil
				}
				return f.replace(targetPath, original, mode)
			})
		}
	}
	return f.replace(targetPath, content, mode)
}

// resolveTarget returns the operating system path for name beneath Root. It
// creates missing parent directories and refuses symbolic links, non-directory
// parents, and non-regular targets.
func (f fileSystem) resolveTarget(name string) (string, error) {
	refuse := &fs.PathError{Op: "write", Path: name, Err: fs.ErrPermission}
	parts := strings.Split(name, "/")
	current := f.Root
	for index, part := range parts {
		current = filepath.Join(current, part)
		last := index == len(parts)-1
		info, err := os.Lstat(current)
		if errors.Is(err, fs.ErrNotExist) {
			if last {
				return current, nil
			}
			if err = os.Mkdir(current, defaultDirectoryMode); err != nil && !errors.Is(err, fs.ErrExist) {
				return "", err
			}
			// Inspect again: another writer may have created the path first.
			info, err = os.Lstat(current)
		}
		if err != nil {
			return "", err
		}
		switch {
		case info.Mode()&fs.ModeSymlink != 0:
			return "", refuse
		case !last && !info.IsDir():
			return "", refuse
		case last && !info.Mode().IsRegular():
			return "", refuse
		}
	}
	return current, nil
}

// snapshot captures the current state of name so that it can be restored.
func (f fileSystem) snapshot(name string) (content []byte, mode fs.FileMode, existed bool, err error) {
	info, err := fs.Stat(f.Directory, name)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, defaultFileMode, false, nil
	}
	if err != nil {
		return nil, 0, false, err
	}
	if !info.Mode().IsRegular() {
		return nil, 0, false, &fs.PathError{Op: "write", Path: name, Err: fs.ErrInvalid}
	}
	if f.transaction != nil {
		if content, err = fs.ReadFile(f.Directory, name); err != nil {
			return nil, 0, false, err
		}
	}
	return content, info.Mode().Perm(), true, nil
}

// replace writes content to a swap file and renames it over targetPath.
func (f fileSystem) replace(targetPath string, content []byte, mode fs.FileMode) (err error) {
	swap := f.Swap
	if swap == nil {
		swap = defaultSwapFileStrategy
	}
	tempPath, tempFile, err := swap(targetPath)
	if err != nil {
		return fmt.Errorf("unable to create swap file: %w", err)
	}
	tempExists := true
	fileClosed := false
	defer func() {
		if !fileClosed {
			_ = tempFile.Close()
		}
		if tempExists {
			_ = os.Remove(tempPath)
		}
	}()

	writer, ok := tempFile.(io.Writer)
	if !ok {
		return fmt.Errorf("swap file %q is not writable", tempPath)
	}
	written, err := writer.Write(content)
	if err == nil && written != len(content) {
		err = io.ErrShortWrite
	}
	if err != nil {
		return fmt.Errorf("unable to write swap file: %w", err)
	}
	if syncer, ok := tempFile.(interface{ Sync() error }); ok {
		if err = syncer.Sync(); err != nil {
			return fmt.Errorf("unable to sync swap file: %w", err)
		}
	}
	fileClosed = true
	if err = tempFile.Close(); err != nil {
		return fmt.Errorf("unable to close swap file: %w", err)
	}
	// Applied by path because fs.File does not expose Chmod; the swap file is
	// not yet visible at the target, so this does not affect atomicity.
	if err = os.Chmod(tempPath, mode); err != nil {
		return fmt.Errorf("unable to preserve target permissions: %w", err)
	}
	if err = os.Rename(tempPath, targetPath); err != nil {
		return err
	}
	tempExists = false
	return nil
}

// fileTransaction records how to undo each file written through it.
type fileTransaction struct {
	mu     sync.Mutex
	closed bool
	seen   map[string]struct{}
	undo   []func() error
}

// Close ends the transaction. If err points to a non-nil error, every file
// written through the transaction is restored to its original state, newest
// first, and any restore failures are joined into *err.
func (t *fileTransaction) Close(err *error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return
	}
	t.closed = true
	undo := t.undo
	t.undo = nil
	if err == nil || *err == nil {
		return
	}
	for i := len(undo) - 1; i >= 0; i-- {
		if restoreErr := undo[i](); restoreErr != nil {
			*err = errors.Join(*err, fmt.Errorf("unable to roll back file change: %w", restoreErr))
		}
	}
}

func defaultSwapFileStrategy(targetPath string) (string, fs.File, error) {
	tempFilePath := "." + filepath.Base(targetPath) + "-*.tmp"
	file, err := os.CreateTemp(
		filepath.Dir(targetPath),
		tempFilePath,
	)
	if err != nil {
		return "", nil, err
	}
	return file.Name(), file, nil
}

type validatedFileSystem struct {
	FileSystem
}
