package cuebook

import (
	"fmt"
	"net/http"
	"strings"

	"cuelang.org/go/cue"
)

// EntryNotFoundError reports that a requested entry does not exist.
type EntryNotFoundError struct {
	Path      string
	ByteRange ByteRange
}

func (e *EntryNotFoundError) Error() string {
	if e.ByteRange == (ByteRange{}) {
		return fmt.Sprintf("entry not found: path=%s", e.Path)
	}
	return fmt.Sprintf("entry not found: path=%s byteRange=%d-%d", e.Path, e.ByteRange.Head, e.ByteRange.Tail)
}

func (e *EntryNotFoundError) HyperTextStatusCode() int {
	return http.StatusNotFound
}

type Entry struct {
	UUID                string
	Title               string
	identifierFieldName string
	Value               cue.Value
	Fields              []Field
	Details             []Field
	FieldDefinitions    []FieldDefinition
}

// UniqueIdentifierFieldName reports the configured schema field that identifies this entry.
func (e Entry) UniqueIdentifierFieldName() string {
	return e.identifierFieldName
}

func (e Entry) GetDescription() (description []string) {
	for _, fields := range [][]Field{e.Fields, e.Details} {
		for _, field := range fields {
			value := field.String()
			if !field.Value.IsConcrete() || strings.TrimSpace(value) == "" {
				value = "..."
			}
			description = append(description, value)
		}
	}
	return description
}

func (e Entry) GetField(atIndex int) (f Field, err error) {
	if atIndex < 0 {
		return f, ErrFieldIndexOutOfRange
	}
	fieldCount := len(e.Fields)
	if atIndex < fieldCount {
		return e.Fields[atIndex], nil
	}
	atIndex -= fieldCount
	if atIndex >= len(e.Details) {
		return f, ErrFieldIndexOutOfRange
	}
	return e.Details[atIndex], nil
}

func (e Entry) GetFieldByName(name string) (field Field, ok bool) {
	for _, field = range e.Fields {
		if field.Name == name {
			return field, true
		}
	}
	for _, field = range e.Details {
		if field.Name == name {
			return field, true
		}
	}
	return
}
