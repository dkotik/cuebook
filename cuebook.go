package cuebook

import (
	"errors"
	"fmt"
	"iter"
	"log/slog"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
)

// Document holds a list of structured data entries.
type Document struct {
	cue.Value
}

func New(source []byte) (book Document, err error) {
	book.Value = cuecontext.New().CompileBytes(source)
	if err = book.Err(); err != nil {
		return book, fmt.Errorf("unable to parse Cue list: %w", err)
	}
	if err = book.Validate(
		cue.Attributes(true),
		cue.Concrete(true),
		cue.Definitions(true),
	); err != nil {
		return book, fmt.Errorf("unable to validate Cue list: %w", err)
	}
	if book.Kind() != cue.ListKind {
		return book, fmt.Errorf("expected a list structued data, got %s", book.Kind())
	}
	return book, nil
}

// TODO: deprecate in favor of cuebook.EachValue
func (d Document) EachValue() iter.Seq[cue.Value] {
	next, err := d.List()
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

func (d Document) EachEntry() iter.Seq2[Entry, error] {
	next, err := d.List()
	if err != nil {
		panic(fmt.Errorf("unable to iterate over the Cue list: %w", err))
	}

	return func(yield func(Entry, error) bool) {
		for next.Next() {
			entry, err := NewEntry(next.Value())
			if !yield(entry, errors.Join(
				err,
				next.Value().Err(),
			)) {
				break
			}
		}
	}
}

func (d Document) GetValue(atIndex int) (cue.Value, error) {
	value := d.LookupPath(cue.MakePath(cue.Index(atIndex)))
	return value, value.Err()
}

func (d Document) GetField(atIndex, fieldIndex int) (f Field, err error) {
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

func (d Document) Len() (int, error) {
	length, err := d.Value.Len().Int64()
	if err != nil {
		panic(fmt.Errorf("unable to get the length of the Cue list: %w", err))
	}
	return int(length), nil
}

func (d Document) LogValue() slog.Value {
	length, _ := d.Len()
	return slog.GroupValue(
		// slog.String("title", d.Metadata().Title()),
		slog.Int("entries", length),
	)
}

func GetByteSpanInSource(v cue.Value) (byteRange SourceByteRange) {
	_, expressions := v.Expr()
	for _, expression := range expressions {
		if expression.IsConcrete() {
			if source := expression.Source(); source != nil {
				// found first concrete data definition
				// with present source pointer
				byteRange.BeginsAt = source.Pos().Offset()
				byteRange.EndsAt = source.End().Offset()
				break
			}
		}
	}
	return byteRange
}
