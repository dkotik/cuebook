package cuebook

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

type Transaction interface {
	Close(*error)
}

type FileSystem interface {
	BeginTransaction(context.Context) Transaction
	WithTransaction(Transaction) (FileSystem, error)
	ReadFile(context.Context, string) ([]byte, error)
	WriteFile(context.Context, string, []byte) error
}

type fileSystem struct {
	Directory fs.FS
	Swap      func(string) (string, io.Reader, error)
}

func replaceFileAtomically(
	targetPath string,
	swapPath string,
	content []byte,
	mode os.FileMode,
	rename func(oldPath, newPath string) error,
) (err error) {
	tempFile, err := os.CreateTemp(swapPath, "."+filepath.Base(targetPath)+"-*.tmp")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
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

	written, writeErr := tempFile.Write(content)
	if writeErr == nil && written != len(content) {
		writeErr = io.ErrShortWrite
	}
	if writeErr != nil {
		return fmt.Errorf("unable to write patch result: %w", writeErr)
	}
	if err = tempFile.Chmod(mode); err != nil {
		return fmt.Errorf("unable to preserve target permissions: %w", err)
	}
	if err = tempFile.Sync(); err != nil {
		return fmt.Errorf("unable to sync patch result: %w", err)
	}
	if err = tempFile.Close(); err != nil {
		fileClosed = true
		return fmt.Errorf("unable to close patch result: %w", err)
	}
	fileClosed = true
	if err = rename(tempPath, targetPath); err != nil {
		return err
	}
	tempExists = false
	return nil
}
