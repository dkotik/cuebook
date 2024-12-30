package cuebook

import (
	"errors"
	"fmt"
	"io"

	"cuelang.org/go/cue"
)

const (
	informationUnavailable = " ⃠"
	booleanTrue            = "✓"
	attrDetail             = "detail"
)

var _ io.WriterTo = (*Entry)(nil)

type Entry struct {
	Title   string
	Fields  []Field
	Details []Field
}

func NewEntry(v cue.Value) (entry Entry, err error) {
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
	var titleFound bool
	for iterator.Next() {
		attr := iterator.Value().Attribute("detail")
		isDetail, _ := attr.Flag(0, attrDetail)
		// if err != nil {
		// 	return entry, fmt.Errorf("unable to read `detail` attribute on structed object field %q: %w", iterator.Selector().String(), err)
		// }
		if isDetail {
			entry.Details = append(entry.Details, Field{
				Name:  iterator.Selector().String(),
				Value: iterator.Value(),
			})
			continue
		}
		value := iterator.Value()
		if !titleFound && value.Kind() == cue.StringKind {
			entry.Title, err = value.String()
			if err != nil {
				return entry, fmt.Errorf("unable to read %q field on structed object: %w", iterator.Selector().String(), err)
			}
			titleFound = true
		}
		entry.Fields = append(entry.Fields, Field{
			Name:  iterator.Selector().String(),
			Value: value,
		})
	}
	if !titleFound {
		entry.Title = informationUnavailable
	}
	return entry, nil
}

func (e *Entry) WriteTo(w io.Writer) (n int64, err error) {
	var written int
	written, err = w.Write([]byte(`  {`))
	n += int64(written)
	if err != nil {
		return n, err
	}

	// for _, field := range e.Fields {
	// 	// TODO: quote name using Cue call
	// 	written, err = fmt.Fprintf(w, "\n    %s: ", field.GetName())
	// 	n += int64(written)
	// 	if err != nil {
	// 		return n, err
	// 	}

	// 	b, err := field.MarshalText()
	// 	if err != nil {
	// 		return n, err
	// 	}
	// 	if len(b) == 0 {
	// 		written, err = w.Write([]byte(`null`))
	// 		n += int64(written)
	// 		if err != nil {
	// 			return n, err
	// 		}
	// 		continue
	// 	}
	// 	written, err = w.Write(b)
	// 	n += int64(written)
	// 	if err != nil {
	// 		return n, err
	// 	}
	// }

	written, err = w.Write([]byte("\n  },\n"))
	n += int64(written)
	if err != nil {
		return n, err
	}
	return 0, nil
}
