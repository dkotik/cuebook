package cuebook

import (
	"errors"
	"fmt"
	"iter"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"github.com/dkotik/cuebook/metadata"
)

type Entry struct {
	title   string
	Value   cue.Value
	Fields  []Field
	Details []Field // TODO: drop fields and details in favor of using iterators
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
			Name:  iterator.Selector().Unquoted(),
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

// EachFieldDefinition emits abstract field definitions for structured entries
// of a Cue list.
func EachFieldDefinition(value cue.Value) iter.Seq2[cue.Selector, cue.Value] {
	return func(yield func(_ cue.Selector, value cue.Value) bool) {
		path := cue.MakePath(cue.Index(0))
		if length, _ := value.Len().Uint64(); length == 0 {
			// insert an empty item, so that field definitions are inheritted
			// from conjuctions; otherwise field query will return cue.BottomKind
			// without any way of enumerating the fields
			value = value.FillPath(path, ast.NewStruct())
		}

		next, _ := value.LookupPath(path).Fields(cue.All())
		for next.Next() {
			if !yield(next.Selector(), next.Value()) {
				break
			}
		}
	}
}
