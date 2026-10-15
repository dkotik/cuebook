package search

import (
	"fmt"

	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// Error identifies a search error code for conditions without additional context.
type Error uint8

const (
	// ErrUnknown is the zero value and represents an unspecified search error.
	ErrUnknown Error = iota
	// ErrSourceFilesystemIsNil occurs when NewFS receives a nil filesystem.
	ErrSourceFilesystemIsNil
	// ErrSearchIndexIsBuilding occurs when a query is issued before initial indexing finishes.
	ErrSearchIndexIsBuilding
	// ErrFileChangeIsNil occurs when ApplyFileChange receives no change callback.
	ErrFileChangeIsNil
)

func (e Error) Error() string {
	switch e {
	case ErrSourceFilesystemIsNil:
		return "search: source filesystem is nil"
	case ErrSearchIndexIsBuilding:
		return "search: index is still being built"
	case ErrFileChangeIsNil:
		return "search: file change is nil"
	default:
		return fmt.Sprintf("unknown search error code #%d", e)
	}
}

func (e Error) LocalizeWith(localizer *i18n.Localizer) (string, error) {
	var message i18n.Message
	switch e {
	case ErrSourceFilesystemIsNil:
		message = i18n.Message{ID: "search.sourceFilesystemIsNil", Other: "search: source filesystem is nil"}
	case ErrSearchIndexIsBuilding:
		message = i18n.Message{ID: "search.indexIsBuilding", Other: "search: index is still being built"}
	case ErrFileChangeIsNil:
		message = i18n.Message{ID: "search.fileChangeIsNil", Other: "search: file change is nil"}
	default:
		message = i18n.Message{ID: "search.unknownError", Other: "unknown search error code #{{.Code}}"}
	}
	return localizeSearchError(localizer, message, map[string]interface{}{"Code": uint8(e)})
}

// ParseError occurs when a source file cannot be parsed while indexing.
type ParseError struct {
	Path  string
	Cause error
}

func (e ParseError) Error() string {
	return fmt.Sprintf("parse CUE file %q for search index: %v", e.Path, e.Cause)
}

func (e ParseError) Unwrap() error { return e.Cause }

func (e ParseError) LocalizeWith(localizer *i18n.Localizer) (string, error) {
	message := i18n.Message{
		ID:    "search.parseCUEFile",
		Other: "parse CUE file {{.Path}} for search index: {{.Err}}",
	}
	return localizeSearchError(localizer, message, map[string]interface{}{"Path": e.Path, "Err": e.Cause})
}

// ReadEntryForIndexError occurs when an entry cannot be read from a parsed source file.
type ReadEntryForIndexError struct {
	Index int
	Path  string
	Err   error
}

func (e ReadEntryForIndexError) Error() string {
	return fmt.Sprintf("read entry %d from %q for search index: %v", e.Index, e.Path, e.Err)
}

func (e ReadEntryForIndexError) Unwrap() error { return e.Err }

func (e ReadEntryForIndexError) LocalizeWith(localizer *i18n.Localizer) (string, error) {
	message := i18n.Message{
		ID:    "search.readEntryForIndex",
		Other: "read entry {{.Index}} from {{.Path}} for search index: {{.Err}}",
	}
	return localizeSearchError(localizer, message, map[string]interface{}{"Index": e.Index, "Path": e.Path, "Err": e.Err})
}

// IndexEntryError occurs when an entry cannot be added to the search index.
type IndexEntryError struct {
	Index int
	Path  string
	Err   error
}

func (e IndexEntryError) Error() string {
	return fmt.Sprintf("index entry %d from %q: %v", e.Index, e.Path, e.Err)
}

func (e IndexEntryError) Unwrap() error { return e.Err }

func (e IndexEntryError) LocalizeWith(localizer *i18n.Localizer) (string, error) {
	message := i18n.Message{
		ID:    "search.indexEntry",
		Other: "index entry {{.Index}} from {{.Path}}: {{.Err}}",
	}
	return localizeSearchError(localizer, message, map[string]interface{}{"Index": e.Index, "Path": e.Path, "Err": e.Err})
}

// WalkSourceFilesystemError occurs when the source filesystem cannot be walked during initial indexing.
type WalkSourceFilesystemError struct {
	Err error
}

func (e WalkSourceFilesystemError) Error() string {
	return fmt.Sprintf("walk source filesystem for search index: %v", e.Err)
}

func (e WalkSourceFilesystemError) Unwrap() error { return e.Err }

func (e WalkSourceFilesystemError) LocalizeWith(localizer *i18n.Localizer) (string, error) {
	message := i18n.Message{
		ID:    "search.walkSourceFilesystem",
		Other: "walk source filesystem for search index: {{.Err}}",
	}
	return localizeSearchError(localizer, message, map[string]interface{}{"Err": e.Err})
}

// IndexEntryKeyError occurs when an indexed result is stored under a non-string key.
type IndexEntryKeyError struct {
	KeyType string
}

func (e IndexEntryKeyError) Error() string {
	return fmt.Sprintf("search index entry key has type %s", e.KeyType)
}

func (e IndexEntryKeyError) LocalizeWith(localizer *i18n.Localizer) (string, error) {
	message := i18n.Message{
		ID:    "search.indexEntryKey",
		Other: "search index entry key has type {{.KeyType}}",
	}
	return localizeSearchError(localizer, message, map[string]interface{}{"KeyType": e.KeyType})
}

func localizeSearchError(localizer *i18n.Localizer, message i18n.Message, data map[string]interface{}) (string, error) {
	return localizer.Localize(&i18n.LocalizeConfig{
		DefaultMessage: &message,
		TemplateData:   data,
	})
}

var _ interface {
	LocalizeWith(*i18n.Localizer) (string, error)
} = Error(0)
