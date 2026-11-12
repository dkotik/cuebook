/*
Package cuebook provides a structured data model and parsing utilities for CUE (CUE Unified Expression) lists.
*/
package cuebook

import (
	"bytes"
	"errors"
	"iter"
	"log/slog"
	"sort"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/dkotik/cuebook/metadata"
	"github.com/dkotik/cuebook/metadata/identifier"
)

type IdentifierGenerator interface {
	GenerateIdentifier() (string, error)
}

type identifierGeneratorFunc func() (string, error)

func (generate identifierGeneratorFunc) GenerateIdentifier() (string, error) {
	return generate()
}

// Book holds a list of structured data entries.
type Book struct {
	Value       cue.Value
	Fields      []Field
	Definitions []FieldDefinition

	source                         []byte
	entryUniqueIdentifierFieldName string
	entryUniqueIdentifierGenerator IdentifierGenerator
}

func New(source []byte) (book Book, err error) {
	book.source = bytes.Clone(source)
	book.Value = cuecontext.New().CompileBytes(book.source)
	if err = book.Value.Err(); err != nil {
		return book, err
	}
	if err = book.Value.Validate(
		cue.Attributes(true),
		cue.Concrete(true),
		cue.Definitions(true),
	); err != nil {
		return book, err
	}
	if book.Value.Kind() != cue.ListKind {
		return book, ErrNotCueList
	}

	fieldSource := firstDisjunct(book.Value)
	for definition := range EachFieldDefinition(fieldSource) {
		book.Definitions = append(book.Definitions, definition)
		book.Fields = append(book.Fields, NewField(definition.Selector.Unquoted(), definition.Value))
	}
	book.configureEntryUniqueIdentifier()
	return book, nil
}

func (book *Book) configureEntryUniqueIdentifier() {
	for _, definition := range book.Definitions {
		attribute, ok := metadata.GetFieldAttributes(definition.Value, "cuebook").GetFirstOf("uuid")
		if !ok {
			continue
		}

		book.entryUniqueIdentifierFieldName = definition.Selector.Unquoted()
		parameters := attribute.Query
		book.entryUniqueIdentifierGenerator = identifierGeneratorFunc(func() (string, error) {
			return identifier.GenerateUUID("", parameters)
		})
		return
	}
}

func (d Book) EachEntry() iter.Seq2[Entry, error] {
	return func(yield func(Entry, error) bool) {
		next, err := d.Value.List()
		if err != nil {
			yield(Entry{}, err)
			return
		}

		fieldOrder := make(map[string]int, len(d.Definitions))
		for index, definition := range d.Definitions {
			name := definition.Selector.Unquoted()
			if _, exists := fieldOrder[name]; !exists {
				fieldOrder[name] = index
			}
		}

		var precedingRange ByteRange
		hasPrecedingRange := false
		for next.Next() {
			value := next.Value()
			entry := Entry{
				Value:                   value,
				FieldDefinitions:        d.Definitions,
				identifierFieldName:     d.entryUniqueIdentifierFieldName,
				source:                  d.source,
				precedingSourceRange:    precedingRange,
				hasPrecedingSourceRange: hasPrecedingRange,
			}
			var entryErr error
			entryRange, rangeErr := NewByteRange(value)
			if rangeErr == nil {
				entry.sourceRange = entryRange
				entry.hasSourceRange = true
				entry.sourceAnchor, rangeErr = anchorForRange(entryRange, d.source)
			}
			if rangeErr == nil {
				precedingRange = entryRange
				hasPrecedingRange = true
			}
			if rangeErr != nil {
				entryErr = errors.Join(entryErr, rangeErr)
			}
			if !value.IsConcrete() || value.IsNull() {
				entryErr = ErrCannotLoadAbstractEntry
			} else if kind := value.Kind(); kind != cue.StructKind {
				entryErr = ErrEntryValueNotStruct
			}

			if entryErr == nil {
				fields, fieldsErr := value.Value().Fields(cue.Optional(true))
				if fieldsErr != nil {
					entryErr = fieldsErr
				} else {
					for fields.Next() {
						fieldValue := fields.Value()
						if metadata.IsTitleField(fieldValue) && fieldValue.IsConcrete() {
							entry.Title = metadata.ValueToString(fieldValue)
						}
						name := fields.Selector().Unquoted()
						if name == d.entryUniqueIdentifierFieldName && fieldValue.IsConcrete() && fieldValue.Kind() == cue.StringKind {
							if identifier, identifierErr := fieldValue.String(); identifierErr == nil && identifier != "" {
								entry.UUID = identifier
							}
						}
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

			if entry.Title == "" {
				for _, field := range entry.Fields {
					if field.Value.IsConcrete() && field.String() != "" {
						entry.Title = field.String()
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
		return Entry{}, ErrBookEntryIndexOutOfRange
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
	return Entry{}, ErrBookEntryIndexOutOfRange
}

func (d Book) GetValue(atIndex int) (cue.Value, error) {
	value := d.Value.LookupPath(cue.MakePath(cue.Index(atIndex)))
	if err := value.Err(); err != nil {
		return value, err
	}
	return value, nil
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
	return int(length), err
}

func (d Book) LogValue() slog.Value {
	length, _ := d.Len()
	return slog.GroupValue(
		// slog.String("title", d.Metadata().Title()),
		slog.Int("entries", length),
	)
}
