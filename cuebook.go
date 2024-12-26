package cuebook

import (
	"errors"
	"fmt"
	"iter"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
)

// CueBook holds a list of maps.
type CueBook struct {
	cue.Value
}

func New(source []byte) (book CueBook, err error) {
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

func (b CueBook) EachValue() iter.Seq[cue.Value] {
	next, err := b.List()
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

func (b CueBook) EachEntry() iter.Seq2[Entry, error] {
	next, err := b.List()
	if err != nil {
		panic(fmt.Errorf("unable to iterate over the Cue list: %w", err))
	}

	return func(yield func(Entry, error) bool) {
		for next.Next() {
			// if err = next.Err(); err != nil {
			// 	return yield(Entry{}, err)
			// }
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

func (b CueBook) Len() int {
	length, err := b.Value.Len().Int64()
	if err != nil {
		panic(fmt.Errorf("unable to get the length of the Cue list: %w", err))
	}
	return int(length)
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
