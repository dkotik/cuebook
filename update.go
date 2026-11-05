package cuebook

import (
	"bytes"
	"fmt"
	"sort"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/literal"
	"cuelang.org/go/cue/parser"
	"cuelang.org/go/cue/token"

	"github.com/dkotik/cuebook/metadata"
)

type replacePatch struct {
	Target      ByteAnchor
	Replacement ByteAnchor
}

func (p replacePatch) Difference() ByteAnchor {
	return p.Replacement
}

func (p replacePatch) ApplyToCueSource(source []byte) ([]byte, error) {
	r, err := p.Target.Match(source)
	if err != nil {
		return nil, err
	}
	return spliceRange(source, r, p.Replacement.Content)
}

func (p replacePatch) Invert() Patch {
	return replacePatch{
		Target:      p.Replacement,
		Replacement: p.Target,
	}
}

func UpdateRange(source []byte, r ByteRange, replacement []byte) (Patch, error) {
	target, err := anchorForRange(r, source)
	if err != nil {
		return nil, err
	}
	return replacePatch{
		Target: target,
		Replacement: ByteAnchor{
			Content:              replacement,
			PreceedingDuplicates: bytes.Count(source[:r.Head], replacement),
		},
	}, nil
}

// entrySyntaxOptions export an entry as pure data. Attributes are omitted
// because the exported value carries the schema's attributes: copying them into
// entries would pin the entry to one schema and break transfers between files
// with similar schemas.
var entrySyntaxOptions = []cue.Option{cue.Concrete(true), cue.Attributes(false)}

// entrySyntax exports an entry for an in-place rewrite. Attributes inherited
// from the schema are omitted, but attributes written in the entry's own source
// are restored so that rewriting one field keeps the author's metadata, such as
// locked or detail markers, on every field.
func entrySyntax(source []byte, entry cue.Value) (*ast.StructLit, error) {
	fields, ok := entry.Syntax(entrySyntaxOptions...).(*ast.StructLit)
	if !ok {
		return nil, ErrEntryNotStruct
	}
	r, err := NewByteRange(entry)
	if err != nil || !validByteRange(r, source) {
		return fields, nil
	}
	written, err := parser.ParseExpr("", source[r.Head:r.Tail])
	if err != nil {
		return fields, nil
	}
	if written, ok := written.(*ast.StructLit); ok {
		restoreWrittenAttributes(fields, written)
	}
	return fields, nil
}

func restoreWrittenAttributes(exported, written *ast.StructLit) {
	var declarations []ast.Decl
	writtenFields := make(map[string]*ast.Field)
	for _, element := range written.Elts {
		switch element := element.(type) {
		case *ast.Attribute:
			declarations = append(declarations, &ast.Attribute{Text: element.Text})
		case *ast.Field:
			if label, _, err := ast.LabelName(element.Label); err == nil {
				writtenFields[label] = element
			}
		}
	}
	for _, element := range exported.Elts {
		field, ok := element.(*ast.Field)
		if !ok {
			continue
		}
		label, _, err := ast.LabelName(field.Label)
		if err != nil {
			continue
		}
		writtenField, ok := writtenFields[label]
		if !ok {
			continue
		}
		// Fresh nodes keep positions from the parsed source out of the output.
		for _, attribute := range writtenField.Attrs {
			field.Attrs = append(field.Attrs, &ast.Attribute{Text: attribute.Text})
		}
		if value, ok := field.Value.(*ast.StructLit); ok {
			if writtenValue, ok := writtenField.Value.(*ast.StructLit); ok {
				restoreWrittenAttributes(value, writtenValue)
			}
		}
	}
	exported.Elts = append(declarations, exported.Elts...)
}

func ReplaceStructListEntry(source []byte, value cue.Value, replacement []byte) (Patch, error) {
	r, err := NewByteRange(value)
	if err != nil {
		return nil, err
	}
	target, err := anchorForRange(r, source)
	if err != nil {
		return nil, err
	}
	return replacePatch{
		Target: target,
		Replacement: ByteAnchor{
			Content:              replacement,
			PreceedingDuplicates: bytes.Count(source[:r.Head], replacement),
		},
	}, nil
}

func UpdateFieldValue(source []byte, entry, field cue.Value, value string) (Patch, error) {
	fields, err := entrySyntax(source, entry)
	if err != nil {
		return nil, err
	}
	search, ok := field.Label()
	if !ok {
		return nil, ErrTargetFieldNotStructField
	}
	replacement, err := metadata.FormatFieldValue(search, field, value)
	if err != nil {
		return nil, err
	}

	found := false
	for _, element := range fields.Elts {
		sourceField, ok := element.(*ast.Field)
		if !ok {
			continue
		}
		label, _, err := ast.LabelName(sourceField.Label)
		if err != nil || label != search {
			continue
		}
		sourceField.Value = replacement.Value
		found = true
		break
	}
	if !found {
		fields.Elts = append(fields.Elts, replacement)
	}

	content, err := format.Node(
		fields,
		format.Simplify(),
		format.IndentPrefix(1),
		format.UseSpaces(4),
	)
	if err != nil {
		return nil, err
	}
	return ReplaceStructListEntry(source, entry, content)
}

// RemoveFieldValue omits a field from an entry's concrete values without
// materializing optional fields or changing the surrounding document.
func RemoveFieldValue(source []byte, entry cue.Value, fieldName string) (Patch, error) {
	fields, err := entrySyntax(source, entry)
	if err != nil {
		return nil, err
	}
	kept := fields.Elts[:0]
	found := false
	for _, element := range fields.Elts {
		if field, ok := element.(*ast.Field); ok {
			label, _, err := ast.LabelName(field.Label)
			if err == nil && label == fieldName {
				found = true
				continue
			}
		}
		kept = append(kept, element)
	}
	if !found {
		return nothingPatch{}, nil
	}
	fields.Elts = kept
	content, err := format.Node(
		fields,
		format.Simplify(),
		format.IndentPrefix(1),
		format.UseSpaces(4),
	)
	if err != nil {
		return nil, err
	}
	return ReplaceStructListEntry(source, entry, content)
}

func MergeFieldValues(source []byte, entry cue.Value, values map[string]string) (_ Patch, err error) {
	fields, err := entrySyntax(source, entry)
	if err != nil {
		return nil, err
	}

	all := make(map[string]cue.Value)
	for selector, field := range eachField(entry, cue.All()) {
		all[selector.Unquoted()] = field
	}
	existing := make(map[string]*ast.Field, len(fields.Elts))
	for _, element := range fields.Elts {
		if field, ok := element.(*ast.Field); ok {
			if label, _, err := ast.LabelName(field.Label); err == nil {
				existing[label] = field
			}
		}
	}

	labels := make([]string, 0, len(values))
	for label := range values {
		labels = append(labels, label)
	}
	sort.Strings(labels)
	for _, label := range labels {
		value := values[label]
		if known, ok := all[label]; ok {
			value, err = metadata.FormatAccordingToAttributes(known, value)
			if err != nil {
				return nil, fmt.Errorf("%w: %w", ErrUnableToFormatFieldValue, err)
			}
		}

		updated := ast.NewLit(
			token.STRING,
			literal.String.
				WithOptionalTabIndent(1).
				Quote(value),
		)
		if field, ok := existing[label]; ok {
			field.Value = updated
			continue
		}
		fields.Elts = append(fields.Elts, &ast.Field{
			Label: ast.NewString(label),
			Value: updated,
		})
	}

	content, err := format.Node(
		fields,
		format.Simplify(),
		format.IndentPrefix(1),
		format.UseSpaces(4),
	)
	if err != nil {
		return nil, err
	}
	return ReplaceStructListEntry(source, entry, content)
}
