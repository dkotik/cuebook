package htmx

import (
	"bytes"
	"html/template"

	"cuelang.org/go/cue"
	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/internal/markdown"
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
	Locked      bool
}

type addIdentifierView struct {
	Value string
}

type entryFieldDefinition struct {
	Field    cuebook.Field
	Optional bool
}

// renderFieldMarkdown converts a field value to safe HTML. Goldmark disables
// raw HTML by default, so source HTML is not included in the returned trusted
// HTML.
func renderFieldMarkdown(value string) template.HTML {
	source := []byte(value)
	var rendered bytes.Buffer
	if err := markdown.NewRenderer().Render(&rendered, source, markdown.NewParser().Parse(source)); err != nil {
		return template.HTML(template.HTMLEscapeString(value))
	}
	return template.HTML(rendered.String())
}

func makeAddFieldViews(document cuebook.Book) ([]addFieldView, *addIdentifierView, error) {
	identifierFieldName, identifier, err := document.GenerateEntryUniqueIdentifier()
	if err != nil {
		return nil, nil, err
	}
	var identifierView *addIdentifierView
	if identifierFieldName != "" {
		identifierView = &addIdentifierView{Value: identifier}
	}

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
			Locked:      metadata.IsLockedField(field.Value),
		}
		switch {
		case kind&cue.NumberKind != 0:
			view.Numeric = true
			if !view.Optional {
				view.Value = "0"
			}
		case kind == cue.BoolKind:
			view.Boolean = true
			view.Value = "not_set"
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
	return result, identifierView, nil
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
	identifierFieldName := document.EntryUniqueIdentifierFieldName()
	allDefinitions := allEntryFieldDefinitions(document)
	if identifierFieldName == "" {
		return allDefinitions
	}

	result := make([]entryFieldDefinition, 0, len(allDefinitions))
	for _, definition := range allDefinitions {
		if definition.Field.Name == identifierFieldName {
			continue
		}
		result = append(result, definition)
	}
	return result
}

func entryIdentifierFieldName(entry cuebook.Entry) string {
	return entry.UniqueIdentifierFieldName()
}

func allEntryFieldDefinitions(document cuebook.Book) []entryFieldDefinition {
	var result []entryFieldDefinition
	entry := document.Value.LookupPath(cue.MakePath(cue.Index(0)))
	for definition := range cuebook.EachFieldDefinition(document.Value) {
		selector := definition.Selector
		result = append(result, entryFieldDefinition{
			Field:    cuebook.NewField(selector.Unquoted(), definition.Value),
			Optional: selector.ConstraintType() == cue.OptionalConstraint || entryFieldOptional(entry, selector.Unquoted()),
		})
	}
	return result
}
