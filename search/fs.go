package search

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"path"
	"sync"
	"sync/atomic"

	"github.com/dkotik/cuebook"
)

type SearchFS interface {
	fs.FS
	IndexReady() bool
	IndexError() error
	Query(string) ([]Result, error)
	UpdateFile(name string) error
	ApplyFileChange(name string, change func() error) error
	RemoveFile(name string) error
}

type searchFS struct {
	source     fs.FS
	index      Index
	updateMu   sync.Mutex
	indexReady atomic.Bool
	indexErr   error
}

func NewFS(source fs.FS) (SearchFS, error) {
	if source == nil {
		return nil, errors.New("search: source filesystem is nil")
	}
	wrapped := &searchFS{
		source: source,
		index:  NewBleveIndex(),
	}
	go wrapped.indexFiles()
	return wrapped, nil
}

func (s *searchFS) Open(name string) (fs.File, error) {
	return s.source.Open(name)
}

func (s *searchFS) IndexReady() bool {
	return s.indexReady.Load()
}

func (s *searchFS) IndexError() error {
	if !s.indexReady.Load() {
		return nil
	}
	return s.indexErr
}

func (s *searchFS) Query(query string) ([]Result, error) {
	if !s.indexReady.Load() {
		return nil, errors.New("search: index is still being built")
	}
	if err := s.IndexError(); err != nil {
		return nil, err
	}
	return s.index.Query(query)
}

func (s *searchFS) UpdateFile(name string) error {
	if !fs.ValidPath(name) {
		return fs.ErrInvalid
	}
	s.updateMu.Lock()
	defer s.updateMu.Unlock()
	return s.updateFile(name)
}

func (s *searchFS) ApplyFileChange(name string, change func() error) error {
	if !fs.ValidPath(name) {
		return fs.ErrInvalid
	}
	if change == nil {
		return errors.New("search: file change is nil")
	}
	s.updateMu.Lock()
	defer s.updateMu.Unlock()
	if err := change(); err != nil {
		return err
	}
	return s.updateFile(name)
}

func (s *searchFS) updateFile(name string) error {
	if path.Ext(name) != ".cue" {
		return s.index.Remove(name)
	}
	source, err := fs.ReadFile(s.source, name)
	if errors.Is(err, fs.ErrNotExist) {
		return s.index.Remove(name)
	}
	if err != nil {
		return err
	}
	if err := s.index.Remove(name); err != nil {
		return err
	}
	document, err := cuebook.New(source)
	if err != nil {
		return fmt.Errorf("parse CUE file %q for search index: %w", name, err)
	}
	index := 0
	for entry, err := range document.EachEntry() {
		if err != nil {
			return fmt.Errorf("read entry %d from %q for search index: %w", index, name, err)
		}
		if err := s.index.Include(name, entry); err != nil {
			return fmt.Errorf("index entry %d from %q: %w", index, name, err)
		}
		index++
	}
	return nil
}

func (s *searchFS) RemoveFile(name string) error {
	if !fs.ValidPath(name) {
		return fs.ErrInvalid
	}
	s.updateMu.Lock()
	defer s.updateMu.Unlock()
	return s.index.Remove(name)
}

func (s *searchFS) indexFiles() {
	defer s.indexReady.Store(true)

	err := fs.WalkDir(s.source, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if name == "." || entry.IsDir() || path.Ext(name) != ".cue" {
			return nil
		}
		if entry.Type()&fs.ModeType != 0 && !entry.Type().IsRegular() {
			return nil
		}
		if err := s.UpdateFile(name); err != nil {
			slog.Debug("skipping CUE file while building search index", "path", name, "error", err)
		}
		return nil
	})
	if err != nil {
		s.indexErr = fmt.Errorf("walk source filesystem for search index: %w", err)
	}
}

var _ fs.FS = (*searchFS)(nil)
