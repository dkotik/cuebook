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
	"github.com/dkotik/cuebook/metadata"
)

// Book holds a list of structured data entries.
type Book struct {
	Value       cue.Value
	Fields      []Field
	Definitions []FieldDefinition
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
	for definition := range EachFieldDefinition(fieldSource) {
		book.Definitions = append(book.Definitions, definition)
		book.Fields = append(book.Fields, NewField(definition.Selector.Unquoted(), definition.Value))
	}
	return book, nil
}

func (d Book) EachEntry() iter.Seq2[Entry, error] {
	next, err := d.Value.List()
	if err != nil {
		panic(fmt.Errorf("unable to iterate over the Cue list: %w", err))
	}

	fieldOrder := make(map[string]int, len(d.Definitions))
	for index, definition := range d.Definitions {
		name := definition.Selector.Unquoted()
		if _, exists := fieldOrder[name]; !exists {
			fieldOrder[name] = index
		}
	}

	return func(yield func(Entry, error) bool) {
		for next.Next() {
			value := next.Value()
			entry := Entry{
				Value:            value,
				FieldDefinitions: d.Definitions,
			}
			var entryErr error
			if !value.IsConcrete() || value.IsNull() {
				entryErr = errors.New("cannot load an abstract value as structured object")
			} else if kind := value.Kind(); kind != cue.StructKind {
				entryErr = fmt.Errorf("value is not a structured object: %s", kind)
			}

			if entryErr == nil {
				fields, fieldsErr := value.Value().Fields(cue.Optional(true))
				if fieldsErr != nil {
					entryErr = fmt.Errorf("unable to iterate through fields of a structured object: %w", fieldsErr)
				} else {
					for fields.Next() {
						fieldValue := fields.Value()
						if metadata.IsTitleField(fieldValue) && fieldValue.IsConcrete() {
							entry.title = metadata.ValueToString(fieldValue)
						}
						name := fields.Selector().Unquoted()
						_, isDefinition := fieldOrder[name]
						if !isDefinition || metadata.IsDetailField(fieldValue) {
							entry.Details = append(entry.Details, NewField(name, fieldValue))
							continue
						}
						entry.Fields = append(entry.Fields, NewField(name, fieldValue))
					}
				}
			}
			if entryErr != nil {
				yield(entry, errors.Join(entryErr, value.Err()))
				return
			}

			if entry.title == "" {
				for _, field := range entry.Fields {
					if field.Value.IsConcrete() && field.String() != "" {
						entry.title = field.String()
						break
					}
				}
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

func (d Book) GetEntry(atIndex int) (Entry, error) {
	if atIndex < 0 {
		return Entry{}, errors.New("not found: index out of range")
	}

	index := 0
	for entry, err := range d.EachEntry() {
		if err != nil {
			return entry, err
		}
		if index == atIndex {
			return entry, nil
		}
		index++
	}
	return Entry{}, errors.New("not found: index out of range")
}

func (d Book) GetValue(atIndex int) (cue.Value, error) {
	value := d.Value.LookupPath(cue.MakePath(cue.Index(atIndex)))
	return value, value.Err()
}

func (d Book) GetField(atIndex, fieldIndex int) (f Field, err error) {
	entry, err := d.GetEntry(atIndex)
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
