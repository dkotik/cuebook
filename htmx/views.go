package htmx

import (
	"cuelang.org/go/cue"
	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/metadata"
)

type addFieldView struct {
	Name      string
	Value     string
	MultiLine bool
	Secret    bool
	Numeric   bool
	Boolean   bool
	Optional  bool
}

type entryFieldDefinition struct {
	Field    cuebook.Field
	Optional bool
}

func makeAddFieldViews(document cuebook.Document) []addFieldView {
	var result []addFieldView
	for _, definition := range entryFieldDefinitions(document) {
		field := definition.Field
		_, secret := metadata.GetFieldAttributes(field.Value, "cuebook").GetFirstOf("argon2id")
		kind := field.Value.IncompleteKind()
		view := addFieldView{
			Name:      field.Name,
			MultiLine: metadata.IsMultiLine(field.Value),
			Secret:    secret,
			Optional:  definition.Optional,
		}
		switch {
		case kind&cue.NumberKind != 0:
			view.Numeric = true
			view.Value = "0"
		case kind&cue.BoolKind != 0:
			view.Boolean = true
			view.Value = "false"
		case kind&cue.ListKind != 0:
			view.MultiLine = true
			view.Value = "[]"
		case kind&cue.StructKind != 0:
			view.MultiLine = true
			view.Value = "{}"
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

func entryFieldDefinitions(document cuebook.Document) []entryFieldDefinition {
	var result []entryFieldDefinition
	for selector, value := range cuebook.EachFieldDefinition(document.Value) {
		result = append(result, entryFieldDefinition{
			Field:    cuebook.Field{Name: selector.Unquoted(), Value: value},
			Optional: selector.ConstraintType() == cue.OptionalConstraint,
		})
	}
	return result
}
