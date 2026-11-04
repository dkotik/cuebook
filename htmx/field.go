package htmx

import (
	"bytes"
	"html/template"

	"cuelang.org/go/cue"
	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/metadata"
	"github.com/yuin/goldmark/v2/parser"
	markdownhtml "github.com/yuin/goldmark/v2/renderer/html"
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
	if err := markdownhtml.New().Render(&rendered, source, parser.New().Parse(source)); err != nil {
		return template.HTML(template.HTMLEscapeString(value))
	}
	return template.HTML(rendered.String())
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
	entry := document.Value.LookupPath(cue.MakePath(cue.Index(0)))
	for selector, value := range cuebook.EachFieldDefinition(document.Value) {
		result = append(result, entryFieldDefinition{
			Field:    cuebook.NewField(selector.Unquoted(), value),
			Optional: selector.ConstraintType() == cue.OptionalConstraint || entryFieldOptional(entry, selector.Unquoted()),
		})
	}
	return result
}
