package cuebook

import (
	"errors"
	"fmt"
	"io"

	"cuelang.org/go/cue"
)

var _ io.WriterTo = (*Entry)(nil)

type Entry struct {
	node *cue.Value

	Fields []FieldType
}

func NewEntry(v cue.Value) (*Entry, error) {
	if !v.IsConcrete() || v.IsNull() {
		return nil, errors.New("cannot load abstract value")
	}
	if k := v.Kind(); k != cue.StructKind {
		return nil, fmt.Errorf("value is not a structured object: %s", k)
	}
	field, err := v.Fields(cue.Concrete(true))
	if err != nil {
		return nil, err
	}

	for field.Next() {
		switch field.Value().Kind() {

		}
	}

	return nil, errors.New("impl")
}

func (e *Entry) WriteTo(w io.Writer) (n int64, err error) {
	var written int
	written, err = w.Write([]byte(`  {`))
	n += int64(written)
	if err != nil {
		return n, err
	}

	for _, field := range e.Fields {
		// TODO: quote name using Cue call
		written, err = fmt.Fprintf(w, "\n    %s: ", field.GetName())
		n += int64(written)
		if err != nil {
			return n, err
		}

		b, err := field.MarshalText()
		if err != nil {
			return n, err
		}
		if len(b) == 0 {
			written, err = w.Write([]byte(`null`))
			n += int64(written)
			if err != nil {
				return n, err
			}
			continue
		}
		written, err = w.Write(b)
		n += int64(written)
		if err != nil {
			return n, err
		}
	}

	written, err = w.Write([]byte("\n  },\n"))
	n += int64(written)
	if err != nil {
		return n, err
	}
	return 0, nil
}

func (e *Entry) GetByteOffsetInSource() (start, end int, ok bool) {
	if e.node != nil {
		_, expressions := e.node.Expr()
		for _, expression := range expressions {
			if expression.IsConcrete() {
				if source := expression.Source(); source != nil {
					// found first concrete data definition
					// with present source pointer
					return source.Pos().Offset(), source.End().Offset(), true
				}
			}
			// ignore abstract definitions
		}
	}
	return 0, 0, false
}
