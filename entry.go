package cuebook

import (
	"errors"
	"fmt"
	"iter"
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
	title            string
	Value            cue.Value
	Fields           []Field
	Details          []Field
	FieldDefinitions []FieldDefinition
}

func (e Entry) GetTitle() string {
	return e.title
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
		return f, errors.New("not found: index out of range") // TODO: model
	}
	fieldCount := len(e.Fields)
	if atIndex < fieldCount {
		return e.Fields[atIndex], nil
	}
	atIndex -= fieldCount
	if atIndex >= len(e.Details) {
		return f, errors.New("not found: index out of range") // TODO: model
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

// EachValue returns an iterator over the elements of a CUE list.
//
// It panics if value cannot be iterated as a list.
func EachValue(value cue.Value) iter.Seq[cue.Value] {
	next, err := value.List()
	if err != nil {
		panic(fmt.Errorf("unable to iterate over the Cue list: %w", err))
	}

	return func(yield func(cue.Value) bool) {
		for next.Next() {
			if !yield(next.Value()) {
				break
			}
		}
	}
}
