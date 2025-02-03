package patch

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/literal"
	"cuelang.org/go/cue/token"
	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/metadata"
)

type replacePatch struct {
	Target      ByteAnchor
	Replacement ByteAnchor
}

func (p replacePatch) Difference() ByteAnchor {
	return p.Replacement
}

func (p replacePatch) ApplyToCueSource(source []byte) (result []byte, err error) {
	r, err := p.Target.Match(source)
	if err != nil {
		return nil, err
	}
	b := &bytes.Buffer{}
	_, _ = io.Copy(b, bytes.NewReader(source[:r.Head]))
	_, _ = io.Copy(b, bytes.NewReader(p.Replacement.Content))
	_, _ = io.Copy(b, bytes.NewReader(source[r.Tail:]))
	return b.Bytes(), nil
}

func (p replacePatch) Invert() Patch {
	return replacePatch{
		Target:      p.Replacement,
		Replacement: p.Target,
	}
}

func UpdateRange(source []byte, r ByteRange, replacement []byte) (Patch, error) {
	return replacePatch{
		Target: r.Anchor(source),
		Replacement: ByteAnchor{
			Content:              replacement,
			PreceedingDuplicates: bytes.Count(source[:r.Head], replacement),
		},
	}, nil
}

func ReplaceStructListEntry(source []byte, value cue.Value, b []byte) (Patch, error) {
	r, err := NewByteRange(value)
	if err != nil {
		return nil, err
	}
	return replacePatch{
		Target: r.Anchor(source),
		Replacement: ByteAnchor{
			Content:              b,
			PreceedingDuplicates: bytes.Count(source[:r.Head], b),
		},
	}, nil
}

func UpdateFieldValue(source []byte, entry, field cue.Value, value string) (Patch, error) {
	tree := entry.Syntax(cue.Concrete(true)) // TODO: concrete OPTION is CRITICAL
	fields, ok := tree.(*ast.StructLit)
	if !ok {
		return nil, errors.New("entry not a struct") // TODO: model error
	}
	search, ok := field.Label()
	if !ok {
		return nil, errors.New("target field not a struct field") // TODO: model error
	}
	iterator, err := entry.Value().Fields(cue.Optional(true))
	if err != nil {
		return nil, fmt.Errorf("unable to iterate through fields of a structured object: %w", err)
	}
	i := 0
	for iterator.Next() {
		label, ok := iterator.Value().Label()
		if !ok {
			return nil, errors.New("source field not a struct field") // TODO: model error
		}
		if label == search {
			fields.Elts[i], err = cuebook.Field{
				Name:  search,
				Value: field,
			}.WithValue(value)
			if err != nil {
				return nil, err
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
		i++
	}
	return nil, errors.New("field not found") // TODO: model error
}

func MergeFieldValues(source []byte, entry cue.Value, values map[string]string) (_ Patch, err error) {
	tree := entry.Syntax(cue.Concrete(true)) // TODO: concrete OPTION is CRITICAL
	fields, ok := tree.(*ast.StructLit)
	if !ok {
		return nil, errors.New("entry not a struct") // TODO: model error
	}

	all := make(map[string]cue.Value)
	concrete := make(map[string]int, len(fields.Elts))
	var (
		label string
		i     int
	)
	for selector, field := range cuebook.EachField(entry, cue.All()) {
		label = selector.Unquoted()
		all[label] = field
		if field.IsConcrete() {
			concrete[label] = i
			i++
		}
	}

	for label, value := range values {
		if known, ok := all[label]; ok {
			value, err = metadata.FormatAccordingToAttributes(known, value)
			if err != nil {
				return nil, fmt.Errorf("failed to format field value: %w", err)
			}
			// if value == "" {
			// 	value, _ = metadata.GetDefaultValue(field)
			// }
			// fmt.Println("sdfsdfsdf:", found.String(), value)
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
