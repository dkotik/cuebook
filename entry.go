package cuebook

import (
	"errors"
	"fmt"
	"iter"

	"cuelang.org/go/cue"
	"github.com/dkotik/cuebook/metadata"
)

type Entry struct {
	title   string
	Value   cue.Value
	Fields  []Field
	Details []Field
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
		if metadata.IsTitleField(value) {
			entry.title = metadata.ValueToString(value)
		}
		if !value.IsConcrete() {
			continue // skip abstract fields
		}
		// attr := value.Attribute("detail")
		// isDetail, _ := attr.Flag(0, attrDetail)
		// if err != nil {
		// 	return entry, fmt.Errorf("unable to read `detail` attribute on structed object field %q: %w", iterator.Selector().String(), err)
		// }
		if metadata.IsDetailField(value) {
			entry.Details = append(entry.Details, Field{
				// Parent: entry,
				Name:  iterator.Selector().String(),
				Value: value,
			})
			// panic(iterator.Selector().String())
			continue
		}
		entry.Fields = append(entry.Fields, Field{
			// Parent: entry,
			Name:  iterator.Selector().String(),
			Value: value,
		})
	}

	if entry.title == "" && len(entry.Fields) > 0 {
		entry.title = entry.Fields[0].String()
	}
	return entry, nil
}

func (e Entry) GetTitle() string {
	return e.title
}

func (e Entry) GetDescription() (description []string) {
	for _, field := range e.Fields {
		description = append(description, field.String())
	}
	if len(description) > 0 && description[0] == e.title {
		return description[1:] // skip first line if identical to title
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

func EachField(value cue.Value, options ...cue.Option) iter.Seq2[cue.Selector, cue.Value] {
	next, err := value.Fields(options...)
	if err != nil {
		panic(fmt.Errorf("unable to iterate over the Cue list: %w", err))
	}

	return func(yield func(cue.Selector, cue.Value) bool) {
		for next.Next() {
			if !yield(next.Selector(), next.Value()) {
				break
			}
		}
	}
}
