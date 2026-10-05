package htmx

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/dkotik/cuebook/patch"
)

// NewDirectory returns a writable handler rooted at directory. Only regular
// .cue files beneath the root are exposed. The root must already exist.
func NewDirectory(directory string) (http.Handler, error) {
	if strings.TrimSpace(directory) == "" {
		return nil, errors.New("htmx: directory is empty")
	}
	root, err := filepath.Abs(directory)
	if err != nil {
		return nil, fmt.Errorf("htmx: resolve directory: %w", err)
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("htmx: resolve directory symlinks: %w", err)
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("htmx: inspect directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("htmx: %q is not a directory", directory)
	}
	return NewWithCommitter(os.DirFS(root), directoryCommitter{root: root})
}

// FileCreator creates a file only when it does not already exist. It is an
// optional capability required for operations that archive entries.
type FileCreator interface {
	CreateFileIfNotExists(name string, content []byte) error
}

type directoryCommitter struct {
	root string
}

func (c directoryCommitter) Commit(name string, change patch.Patch) error {
	if !validFileName(name) {
		return fs.ErrPermission
	}
	target := filepath.Join(c.root, filepath.FromSlash(name))
	relative, err := filepath.Rel(c.root, target)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fs.ErrPermission
	}
	if err := c.verifyRegularFile(name); err != nil {
		return err
	}
	_, err = patch.Commit(target, filepath.Dir(target), change)
	return err
}

func (c directoryCommitter) CreateFileIfNotExists(name string, content []byte) error {
	if !validFileName(name) {
		return fs.ErrPermission
	}

	parts := strings.Split(name, "/")
	current := c.root
	for _, part := range parts[:len(parts)-1] {
		current = filepath.Join(current, part)
		if err := os.Mkdir(current, 0o755); err != nil && !errors.Is(err, fs.ErrExist) {
			return err
		}
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fs.ErrPermission
		}
	}

	target := filepath.Join(c.root, filepath.FromSlash(name))
	file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, fs.ErrExist) {
		return c.verifyRegularFile(name)
	}
	if err != nil {
		return err
	}

	written, writeErr := file.Write(content)
	if writeErr == nil && written != len(content) {
		writeErr = io.ErrShortWrite
	}
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		_ = os.Remove(target)
		return errors.Join(writeErr, closeErr)
	}
	return nil
}

func (c directoryCommitter) verifyRegularFile(name string) error {
	current := c.root
	parts := strings.Split(name, "/")
	for index, part := range parts {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fs.ErrPermission
		}
		if index < len(parts)-1 && !info.IsDir() {
			return fs.ErrPermission
		}
		if index == len(parts)-1 && !info.Mode().IsRegular() {
			return fs.ErrPermission
		}
	}
	return nil
}
