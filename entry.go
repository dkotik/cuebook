package cuebook

import (
	"errors"
	"fmt"
	"iter"
	"net/http"
	"strings"

	"cuelang.org/go/cue"
	fieldmodel "github.com/dkotik/cuebook/htmx/field"
	"github.com/dkotik/cuebook/metadata"
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
	title   string
	Value   cue.Value
	Fields  []fieldmodel.Field
	Details []fieldmodel.Field // TODO: drop fields and details in favor of using iterators
}

func NewEntry(v cue.Value) (entry Entry, err error) {
	entry.Value = v
	if !v.IsConcrete() || v.IsNull() {
		return entry, errors.New("cannot load an abstract value as structured object")
	}
	if k := v.Kind(); k != cue.StructKind {
		return entry, fmt.Errorf("value is not a structured object: %s", k)
	}

	iterator, err := v.Value().Fields(cue.Optional(true))
	if err != nil {
		return entry, fmt.Errorf("unable to iterate through fields of a structured object: %w", err)
	}
	for iterator.Next() {
		value := iterator.Value()
		if metadata.IsTitleField(value) && value.IsConcrete() {
			entry.title = metadata.ValueToString(value)
		}
		if metadata.IsDetailField(value) {
			entry.Details = append(entry.Details, fieldmodel.NewField(iterator.Selector().Unquoted(), value))
			continue
		}
		entry.Fields = append(entry.Fields, fieldmodel.NewField(iterator.Selector().Unquoted(), value))
	}

	if entry.title == "" {
		for _, field := range entry.Fields {
			if field.Value.IsConcrete() && field.String() != "" {
				entry.title = field.String()
				break
			}
		}
	}
	return entry, nil
}

func (e Entry) GetTitle() string {
	return e.title
}

func (e Entry) GetDescription() (description []string) {
	for _, fields := range [][]fieldmodel.Field{e.Fields, e.Details} {
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

func (e Entry) GetField(atIndex int) (f fieldmodel.Field, err error) {
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

func (e Entry) GetFieldByName(name string) (field fieldmodel.Field, ok bool) {
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
