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
		view := addFieldView{
			Name:      field.Name,
			MultiLine: metadata.IsMultiLine(field.Value),
			Secret:    secret,
			Optional:  definition.Optional,
		}
		if !secret {
			if value, ok := field.Default(); ok {
				view.Value = value
			} else if field.Value.IsConcrete() {
				view.Value = field.String()
			}
		}
		if field.Value.IncompleteKind()&cue.StringKind == 0 && field.Value.IncompleteKind()&(cue.ListKind|cue.StructKind) != 0 {
			view.MultiLine = true
		}
		result = append(result, view)
	}
	return result
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
