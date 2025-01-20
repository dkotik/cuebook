package cuebook

import (
	"errors"
	"fmt"
	"iter"

	"cuelang.org/go/cue"
)

const (
	informationUnavailable = " ⃠"
	booleanTrue            = "✓"
	attrDetail             = "detail"
)

type Entry struct {
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
		if !value.IsConcrete() {
			continue // skip abstract fields
		}
		attr := value.Attribute("detail")
		isDetail, _ := attr.Flag(0, attrDetail)
		// if err != nil {
		// 	return entry, fmt.Errorf("unable to read `detail` attribute on structed object field %q: %w", iterator.Selector().String(), err)
		// }
		if isDetail {
			entry.Details = append(entry.Details, Field{
				Parent: entry,
				Name:   iterator.Selector().String(),
				Value:  value,
			})
			continue
		}
		entry.Fields = append(entry.Fields, Field{
			Parent: entry,
			Name:   iterator.Selector().String(),
			Value:  value,
		})
	}
	return entry, nil
}

func (e Entry) GetTitle() string {
	if len(e.Fields) > 0 {
		return e.Fields[0].String()
	}
	return informationUnavailable
}

func (e Entry) GetDescription() (description []string) {
	if len(e.Fields) > 1 {
		for _, field := range e.Fields[1:] {
			description = append(description, field.String())
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
