package cuebook

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// FileSystemError identifies a filesystem failure with a fixed message.
type FileSystemError uint8

const (
	ErrUnknownFileSystemMessage FileSystemError = iota
	ErrFileSystemRootEmpty
	ErrFileSystemNotDirectory
	ErrFileSystemSwapFileNotWritable
	ErrFileSystemTransactionClosed
	_ // Retains subsequent FileSystemError values after removing unused swap-file creation code.
	_ // Retains subsequent FileSystemError values after removing unused write code.
	_ // Retains subsequent FileSystemError values after removing unused sync code.
	_ // Retains subsequent FileSystemError values after removing unused close code.
	_ // Retains subsequent FileSystemError values after removing unused permission-preservation code.
	_ // Retains subsequent FileSystemError values after removing unused rollback code.
	// ErrFileSystemFileNotFound reports when a requested file does not exist.
	ErrFileSystemFileNotFound
	_ // Retains subsequent FileSystemError values after removing field iteration.
	_ // Retains subsequent FileSystemError values after removing unused CUE parsing code.
)

func (e FileSystemError) Error() string {
	switch e {
	case ErrFileSystemRootEmpty:
		return "file system root is empty"
	case ErrFileSystemNotDirectory:
		return "not a directory"
	case ErrFileSystemSwapFileNotWritable:
		return "swap file is not writable"
	case ErrFileSystemTransactionClosed:
		return "transaction is closed"
	case ErrFileSystemFileNotFound:
		return "file not found"
	default:
		return fmt.Sprintf("unknown file system message error #%d", e)
	}
}

func (e FileSystemError) Unwrap() error {
	switch e {
	case ErrFileSystemNotDirectory, ErrFileSystemSwapFileNotWritable:
		return fs.ErrInvalid
	case ErrFileSystemFileNotFound:
		return fs.ErrNotExist
	default:
		return nil
	}
}

// HyperTextStatusCode returns 404 for a missing file and 500 for other filesystem errors.
func (e FileSystemError) HyperTextStatusCode() int {
	if e == ErrFileSystemFileNotFound {
		return http.StatusNotFound
	}
	return http.StatusInternalServerError
}

func (e FileSystemError) LocalizeWith(localizer *i18n.Localizer) (string, error) {
	var message i18n.Message
	switch e {
	case ErrFileSystemRootEmpty:
		message = i18n.Message{ID: "filesystem.rootEmpty", Other: "file system root is empty"}
	case ErrFileSystemNotDirectory:
		message = i18n.Message{ID: "filesystem.notDirectory", Other: "not a directory"}
	case ErrFileSystemSwapFileNotWritable:
		message = i18n.Message{ID: "filesystem.swapFileNotWritable", Other: "swap file is not writable"}
	case ErrFileSystemTransactionClosed:
		message = i18n.Message{ID: "filesystem.transactionClosed", Other: "transaction is closed"}
	case ErrFileSystemFileNotFound:
		message = i18n.Message{ID: "filesystem.fileNotFound", Other: "file not found"}
	default:
		message = i18n.Message{ID: "filesystem.unknownMessage", Other: "unknown file system message error #{{.Code}}"}
	}
	return localizer.Localize(&i18n.LocalizeConfig{
		DefaultMessage: &message,
		TemplateData:   map[string]any{"Code": uint8(e)},
	})
}

// UnsupportedTransactionTypeError reports a transaction that is not supported
// by this filesystem implementation.
type UnsupportedTransactionTypeError struct {
	TransactionType string
}

func (e *UnsupportedTransactionTypeError) Error() string {
	return fmt.Sprintf("unsupported transaction type %s", e.TransactionType)
}

func (e *UnsupportedTransactionTypeError) LocalizeWith(localizer *i18n.Localizer) (string, error) {
	message := i18n.Message{ID: "filesystem.unsupportedTransactionType", Other: "unsupported transaction type {{.TransactionType}}"}
	return localizer.Localize(&i18n.LocalizeConfig{
		DefaultMessage: &message,
		TemplateData:   map[string]any{"TransactionType": e.TransactionType},
	})
}

// FileSystemPathError reports a filesystem failure while retaining its underlying
// cause and, when available, path operation details.
type FileSystemPathError struct {
	*fs.PathError
}

func (e *FileSystemPathError) Error() string {
	if e == nil || e.PathError == nil {
		return "file system error"
	}
	if e.Op == "" && e.Path == "" {
		if e.Err == nil {
			return "file system error"
		}
		return e.Err.Error()
	}
	if errors.Is(e.Err, ErrFileSystemFileNotFound) {
		return fmt.Sprintf("file not found: path=%s", e.Path)
	}
	if errors.Is(e.Err, ErrFileSystemSwapFileNotWritable) {
		return fmt.Sprintf("swap file %q is not writable", e.Path)
	}
	return e.PathError.Error()
}

func (e *FileSystemPathError) Unwrap() error {
	if e == nil || e.PathError == nil {
		return nil
	}
	return e.PathError
}

func (e *FileSystemPathError) LocalizeWith(localizer *i18n.Localizer) (string, error) {
	if e == nil || e.PathError == nil {
		message := i18n.Message{ID: "filesystem.error", Other: "file system error"}
		return localizer.Localize(&i18n.LocalizeConfig{DefaultMessage: &message})
	}
	localizedCause, err := localizeFileSystemCause(localizer, e.Err)
	if err != nil {
		return "", err
	}
	if e.Op == "" && e.Path == "" {
		return localizedCause, nil
	}
	if errors.Is(e.Err, ErrFileSystemFileNotFound) {
		message := i18n.Message{ID: "filesystem.fileNotFound", Other: "file not found: path={{.Path}}"}
		return localizer.Localize(&i18n.LocalizeConfig{
			DefaultMessage: &message,
			TemplateData:   map[string]any{"Path": e.Path},
		})
	}
	if errors.Is(e.Err, ErrFileSystemSwapFileNotWritable) {
		message := i18n.Message{ID: "filesystem.swapFileNotWritable", Other: "swap file {{printf \"%q\" .Path}} is not writable"}
		return localizer.Localize(&i18n.LocalizeConfig{
			DefaultMessage: &message,
			TemplateData:   map[string]any{"Path": e.Path},
		})
	}
	message := i18n.Message{ID: "filesystem.pathError", Other: "{{.Op}} {{.Path}}: {{.Cause}}"}
	return localizer.Localize(&i18n.LocalizeConfig{
		DefaultMessage: &message,
		TemplateData:   map[string]any{"Op": e.Op, "Path": e.Path, "Cause": localizedCause},
	})
}

func localizeFileSystemCause(localizer *i18n.Localizer, cause error) (string, error) {
	if cause == nil {
		return "", nil
	}
	if localized, ok := cause.(Localizable); ok {
		return localized.LocalizeWith(localizer)
	}

	if joined, ok := cause.(interface{ Unwrap() []error }); ok {
		causes := joined.Unwrap()
		messages := make([]string, 0, len(causes))
		for _, item := range causes {
			message, err := localizeFileSystemCause(localizer, item)
			if err != nil {
				return "", err
			}
			messages = append(messages, message)
		}
		return strings.Join(messages, "\n"), nil
	}
	return cause.Error(), nil
}

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
		return nil, &FileSystemPathError{PathError: &fs.PathError{Err: ErrFileSystemRootEmpty}}
	}
	root, err := filepath.Abs(config.Root)
	if err != nil {
		return nil, &FileSystemPathError{PathError: &fs.PathError{Err: err}}
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, &FileSystemPathError{PathError: &fs.PathError{Err: err}}
	}
	if !info.IsDir() {
		return nil, &FileSystemPathError{PathError: &fs.PathError{
			Op: "open", Path: root, Err: ErrFileSystemNotDirectory,
		}}
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
		return nil, &FileSystemPathError{PathError: &fs.PathError{
			Err: &UnsupportedTransactionTypeError{TransactionType: fmt.Sprintf("%T", t)},
		}}
	}
	tx.mu.Lock()
	defer tx.mu.Unlock()
	if tx.closed {
		return nil, ErrFileSystemTransactionClosed
	}
	f.transaction = tx
	return f, nil
}

// ReadFile reads the named file from Directory.
func (f fileSystem) ReadFile(ctx context.Context, name string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, &FileSystemPathError{PathError: &fs.PathError{Err: err}}
	}
	content, err := fs.ReadFile(f.Directory, name)
	if err != nil {
		return nil, &FileSystemPathError{PathError: &fs.PathError{Err: err}}
	}
	return content, nil
}

// WriteFile atomically replaces the named file with content: content is
// written to a swap file, which is then moved over the target with
// [os.Rename]. Existing permissions are preserved. Missing parent directories
// are created, and paths through symbolic links are refused so that writes
// cannot escape Root.
func (f fileSystem) WriteFile(ctx context.Context, name string, content []byte) error {
	if err := ctx.Err(); err != nil {
		return &FileSystemPathError{PathError: &fs.PathError{Err: err}}
	}
	if !fs.ValidPath(name) || name == "." {
		return &FileSystemPathError{PathError: &fs.PathError{
			Op: "write", Path: name, Err: fs.ErrInvalid,
		}}
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
			return ErrFileSystemTransactionClosed
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
				return "", &FileSystemPathError{PathError: &fs.PathError{Err: err}}
			}
			// Inspect again: another writer may have created the path first.
			info, err = os.Lstat(current)
		}
		if err != nil {
			return "", &FileSystemPathError{PathError: &fs.PathError{Err: err}}
		}
		switch {
		case info.Mode()&fs.ModeSymlink != 0:
			return "", &FileSystemPathError{PathError: &fs.PathError{
				Op: "write", Path: name, Err: fs.ErrPermission,
			}}
		case !last && !info.IsDir():
			return "", &FileSystemPathError{PathError: &fs.PathError{
				Op: "write", Path: name, Err: fs.ErrPermission,
			}}
		case last && !info.Mode().IsRegular():
			return "", &FileSystemPathError{PathError: &fs.PathError{
				Op: "write", Path: name, Err: fs.ErrPermission,
			}}
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
		return nil, 0, false, &FileSystemPathError{PathError: &fs.PathError{Err: err}}
	}
	if !info.Mode().IsRegular() {
		return nil, 0, false, &FileSystemPathError{PathError: &fs.PathError{
			Op: "write", Path: name, Err: fs.ErrInvalid,
		}}
	}
	if f.transaction != nil {
		if content, err = fs.ReadFile(f.Directory, name); err != nil {
			return nil, 0, false, &FileSystemPathError{PathError: &fs.PathError{Err: err}}
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
		return &FileSystemPathError{PathError: &fs.PathError{
			Err: err,
		}}
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
		return &FileSystemPathError{PathError: &fs.PathError{
			Op: "write", Path: tempPath, Err: ErrFileSystemSwapFileNotWritable,
		}}
	}
	written, err := writer.Write(content)
	if err == nil && written != len(content) {
		err = io.ErrShortWrite
	}
	if err != nil {
		return &FileSystemPathError{PathError: &fs.PathError{
			Err: err,
		}}
	}
	if syncer, ok := tempFile.(interface{ Sync() error }); ok {
		if err = syncer.Sync(); err != nil {
			return &FileSystemPathError{PathError: &fs.PathError{
				Err: err,
			}}
		}
	}
	fileClosed = true
	if err = tempFile.Close(); err != nil {
		return &FileSystemPathError{PathError: &fs.PathError{
			Err: err,
		}}
	}
	// Applied by path because fs.File does not expose Chmod; the swap file is
	// not yet visible at the target, so this does not affect atomicity.
	if err = os.Chmod(tempPath, mode); err != nil {
		return &FileSystemPathError{PathError: &fs.PathError{
			Err: err,
		}}
	}
	if err = os.Rename(tempPath, targetPath); err != nil {
		return &FileSystemPathError{PathError: &fs.PathError{Err: err}}
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
			rollbackErr := &FileSystemPathError{PathError: &fs.PathError{
				Err: restoreErr,
			}}
			*err = &FileSystemPathError{PathError: &fs.PathError{Err: errors.Join(*err, rollbackErr)}}
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
		return "", nil, &FileSystemPathError{PathError: &fs.PathError{Err: err}}
	}
	return file.Name(), file, nil
}

type validatedFileSystem struct {
	FileSystem
}
