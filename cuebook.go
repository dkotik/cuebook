/*
Package cuebook provides a structured data model and parsing utilities for CUE (CUE Unified Expression) lists.
*/
package cuebook

import (
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"sort"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	fieldmodel "github.com/dkotik/cuebook/htmx/field"
)

// Book holds a list of structured data entries.
type Book struct {
	Value  cue.Value
	Fields []Field
}

func New(source []byte) (book Book, err error) {
	book.Value = cuecontext.New().CompileBytes(source)
	if err = book.Value.Err(); err != nil {
		return book, fmt.Errorf("unable to parse Cue list: %w", err)
	}
	if err = book.Value.Validate(
		cue.Attributes(true),
		cue.Concrete(true),
		cue.Definitions(true),
	); err != nil {
		return book, fmt.Errorf("unable to validate Cue list: %w", err)
	}
	if book.Value.Kind() != cue.ListKind {
		return book, fmt.Errorf("expected a list structued data, got %s", book.Value.Kind())
	}

	fieldSource := firstDisjunct(book.Value)
	for selector, value := range EachFieldDefinition(fieldSource) {
		book.Fields = append(book.Fields, NewField(selector.Unquoted(), value))
	}
	return book, nil
}

func (d Book) EachEntry() iter.Seq2[Entry, error] {
	next, err := d.Value.List()
	if err != nil {
		panic(fmt.Errorf("unable to iterate over the Cue list: %w", err))
	}

	fieldOrder := make(map[string]int, len(d.Fields))
	for index, field := range d.Fields {
		if _, exists := fieldOrder[field.Name]; !exists {
			fieldOrder[field.Name] = index
		}
	}

	return func(yield func(Entry, error) bool) {
		for next.Next() {
			entry, err := NewEntry(next.Value())
			if err != nil {
				yield(entry, errors.Join(
					err,
					next.Value().Err(),
				))
				return
			}

			sort.SliceStable(entry.Fields, func(i, j int) bool {
				leftOrder, leftKnown := fieldOrder[entry.Fields[i].Name]
				rightOrder, rightKnown := fieldOrder[entry.Fields[j].Name]
				if leftKnown != rightKnown {
					return leftKnown
				}
				return leftKnown && leftOrder < rightOrder
			})

			if !yield(entry, nil) {
				break
			}
		}
	}
}

func (d Book) GetValue(atIndex int) (cue.Value, error) {
	value := d.Value.LookupPath(cue.MakePath(cue.Index(atIndex)))
	return value, value.Err()
}

func (d Book) GetField(atIndex, fieldIndex int) (f fieldmodel.Field, err error) {
	value, err := d.GetValue(atIndex)
	if err != nil {
		return
	}
	entry, err := NewEntry(value)
	if err != nil {
		return
	}
	return entry.GetField(fieldIndex)
}

func (d Book) Len() (int, error) {
	length, err := d.Value.Len().Int64()
	if err != nil {
		panic(fmt.Errorf("unable to get the length of the Cue list: %w", err))
	}
	return int(length), nil
}

func (d Book) LogValue() slog.Value {
	length, _ := d.Len()
	return slog.GroupValue(
		// slog.String("title", d.Metadata().Title()),
		slog.Int("entries", length),
	)
}
