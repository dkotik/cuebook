package cuebook

import (
	"bytes"
	"fmt"
	"sort"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/literal"
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
	tree := entry.Syntax(cue.Concrete(true))
	fields, ok := tree.(*ast.StructLit)
	if !ok {
		return nil, ErrEntryNotStruct
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
	fields, ok := entry.Syntax(cue.Concrete(true)).(*ast.StructLit)
	if !ok {
		return nil, ErrEntryNotStruct
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
	tree := entry.Syntax(cue.Concrete(true))
	fields, ok := tree.(*ast.StructLit)
	if !ok {
		return nil, ErrEntryNotStruct
	}

	all := make(map[string]cue.Value)
	concrete := make(map[string]int, len(fields.Elts))
	var (
		label string
		i     int
	)
	for selector, field := range EachField(entry, cue.All()) {
		label = selector.Unquoted()
		all[label] = field
		if field.IsConcrete() {
			concrete[label] = i
			i++
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

		updated := &ast.Field{
			Label: ast.NewString(label),
			Value: ast.NewLit(
				token.STRING,
				literal.String.
					WithOptionalTabIndent(1).
					Quote(value),
			),
		}

		index, ok := concrete[label]
		if ok {
			fields.Elts[index] = updated
			continue
		}
		fields.Elts = append(fields.Elts, updated)
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
