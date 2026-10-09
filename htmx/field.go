package htmx

import (
	"cuelang.org/go/cue"
	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/metadata"
)

type addFieldView struct {
	Index       int
	Name        string
	Description string
	Value       string
	MultiLine   bool
	Secret      bool
	Numeric     bool
	Boolean     bool
	Optional    bool
}

type entryFieldDefinition struct {
	Field    cuebook.Field
	Optional bool
}

func makeAddFieldViews(document cuebook.Book) []addFieldView {
	var result []addFieldView
	for index, definition := range entryFieldDefinitions(document) {
		field := definition.Field
		_, secret := metadata.GetFieldAttributes(field.Value, "cuebook").GetFirstOf("argon2id")
		kind := field.Value.IncompleteKind()
		view := addFieldView{
			Index:       index,
			Name:        field.Name,
			Description: field.Description,
			MultiLine:   metadata.IsMultiLine(field.Value),
			Secret:      secret,
			Optional:    definition.Optional,
		}
		switch {
		case kind&cue.NumberKind != 0:
			view.Numeric = true
			if !view.Optional {
				view.Value = "0"
			}
		case kind&cue.BoolKind != 0:
			view.Boolean = true
			if !view.Optional {
				view.Value = "false"
			}
		case kind&cue.ListKind != 0:
			view.MultiLine = true
			if !view.Optional {
				view.Value = "[]"
			}
		case kind&cue.StructKind != 0:
			view.MultiLine = true
			if !view.Optional {
				view.Value = "{}"
			}
		}
		result = append(result, view)
	}
	return result
}

func splitAddFieldViews(fields []addFieldView) (required, optional []addFieldView) {
	for _, field := range fields {
		if field.Optional {
			optional = append(optional, field)
			continue
		}
		required = append(required, field)
	}
	return required, optional
}

func entryFieldDefinitions(document cuebook.Book) []entryFieldDefinition {
	var result []entryFieldDefinition
	for selector, value := range cuebook.EachFieldDefinition(document.Value) {
		result = append(result, entryFieldDefinition{
			Field:    cuebook.NewField(selector.Unquoted(), value),
			Optional: selector.ConstraintType() == cue.OptionalConstraint,
		})
	}
	return result
}
