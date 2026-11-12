package cuebook

import (
	"bytes"
	"errors"
	"sort"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/literal"
	"cuelang.org/go/cue/parser"
	"cuelang.org/go/cue/token"

	"github.com/dkotik/cuebook/metadata"
)

type replacePatch struct {
	Target                 ByteAnchor
	Replacement            ByteAnchor
	DifferenceAnchor       ByteAnchor
	BeforeDifferenceAnchor ByteAnchor
}

func (p replacePatch) Difference() ByteAnchor {
	if !isZeroByteAnchor(p.DifferenceAnchor) {
		return cloneByteAnchor(p.DifferenceAnchor)
	}
	return cloneByteAnchor(p.Replacement)
}

func isZeroByteAnchor(anchor ByteAnchor) bool {
	return len(anchor.Content) == 0 && len(anchor.Prefix) == 0 && len(anchor.Suffix) == 0 && anchor.PreceedingDuplicates == 0
}

func (p replacePatch) ApplyToCueSource(source []byte) ([]byte, error) {
	before, beforeErr := p.Target.MatchUnique(source)
	after, afterErr := p.Replacement.MatchUnique(source)

	if beforeErr == nil && afterErr == nil {
		if before == after && bytes.Equal(p.Target.Content, p.Replacement.Content) {
			return source, nil
		}
		return nil, ErrPatchTargetAmbiguous
	}
	if beforeErr == nil {
		if errors.Is(afterErr, ErrPatchTargetAmbiguous) {
			return nil, ErrPatchTargetAmbiguous
		}
		return spliceRange(source, before, p.Replacement.Content)
	}
	if afterErr == nil {
		return source, nil
	}
	if errors.Is(beforeErr, ErrPatchTargetAmbiguous) || errors.Is(afterErr, ErrPatchTargetAmbiguous) {
		return nil, ErrPatchTargetAmbiguous
	}
	return nil, ErrPatchTargetConflict
}

func (p replacePatch) Invert() Patch {
	beforeDifference := p.BeforeDifferenceAnchor
	if isZeroByteAnchor(beforeDifference) {
		beforeDifference = p.Target
	}
	return replacePatch{
		Target:                 cloneByteAnchor(p.Replacement),
		Replacement:            cloneByteAnchor(p.Target),
		DifferenceAnchor:       cloneByteAnchor(beforeDifference),
		BeforeDifferenceAnchor: cloneByteAnchor(p.Difference()),
	}
}

// UpdateRange creates a patch that replaces the byte range identified by target.
// The anchor must have been created from the source containing the range.
func UpdateRange(target ByteAnchor, replacement []byte) (Patch, error) {
	if len(target.Content) == 0 && target.PreceedingDuplicates == 0 && len(target.Prefix) == 0 && len(target.Suffix) == 0 {
		return nil, ErrByteRangeNotFound
	}
	target = cloneByteAnchor(target)
	after := cloneByteAnchor(target)
	after.Content = bytes.Clone(replacement)
	return replacePatch{
		Target:                 target,
		Replacement:            after,
		DifferenceAnchor:       cloneByteAnchor(after),
		BeforeDifferenceAnchor: cloneByteAnchor(target),
	}, nil
}

// entrySyntaxOptions export an entry as pure data. Attributes are omitted
// because the exported value carries the schema's attributes: copying them into
// entries would pin the entry to one schema and break transfers between files
// with similar schemas.
var entrySyntaxOptions = []cue.Option{cue.Concrete(true), cue.Attributes(false)}

func entrySyntaxFromWrittenSource(entry cue.Value, written []byte) (*ast.StructLit, error) {
	fields, ok := entry.Syntax(entrySyntaxOptions...).(*ast.StructLit)
	if !ok {
		return nil, ErrEntryNotStruct
	}
	expression, err := parser.ParseExpr("", written)
	if err != nil {
		return fields, nil
	}
	if writtenFields, ok := expression.(*ast.StructLit); ok {
		restoreWrittenAttributes(fields, writtenFields)
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

// ReplaceStructListEntry creates a patch that replaces this entry's source.
func (e Entry) ReplaceStructListEntry(replacement []byte) (Patch, error) {
	r, err := e.originalSourceRange()
	if err != nil {
		return nil, err
	}
	if e.identifierFieldName != "" {
		if err := validateIdentifierReplacement(e, replacement); err != nil {
			return nil, err
		}
	}
	return entryReplacementPatch(e, r, bytes.Clone(replacement))
}

func validateIdentifierReplacement(entry Entry, replacement []byte) error {
	path := cue.MakePath(cue.Str(entry.identifierFieldName))
	before, beforePresent, err := stringFieldState(entry.Value.LookupPath(path))
	if err != nil {
		return ErrEntryIdentifierImmutable
	}
	replacementValue := cuecontext.New().CompileBytes(replacement)
	if err := replacementValue.Err(); err != nil {
		return err
	}
	after, afterPresent, err := stringFieldState(replacementValue.LookupPath(path))
	if err != nil || beforePresent != afterPresent || before != after {
		return ErrEntryIdentifierImmutable
	}
	return nil
}

func stringFieldState(value cue.Value) (string, bool, error) {
	if !value.Exists() {
		return "", false, nil
	}
	if !value.IsConcrete() || value.Kind() != cue.StringKind {
		return "", true, ErrEntryIdentifierImmutable
	}
	text, err := value.String()
	return text, true, err
}

func replaceByteRange(source []byte, r ByteRange, replacement []byte) (Patch, error) {
	return replaceByteRangeWithDifference(source, r, replacement, ByteAnchor{})
}

func replaceByteRangeWithDifference(source []byte, r ByteRange, replacement []byte, difference ByteAnchor) (replacePatch, error) {
	target, err := anchorForRange(r, source)
	if err != nil {
		return replacePatch{}, err
	}
	after := cloneByteAnchor(target)
	after.Content = bytes.Clone(replacement)
	after.PreceedingDuplicates = bytes.Count(source[:r.Head], replacement)
	if isZeroByteAnchor(difference) {
		difference = after
	}
	return replacePatch{
		Target:                 target,
		Replacement:            after,
		DifferenceAnchor:       cloneByteAnchor(difference),
		BeforeDifferenceAnchor: cloneByteAnchor(target),
	}, nil
}

// UpdateFieldValue creates a patch that changes a field in this entry.
func (e Entry) UpdateFieldValue(fieldName, value string) (Patch, error) {
	if e.Value.IncompleteKind() != cue.StructKind {
		return nil, ErrEntryNotStruct
	}
	field, exists := e.GetFieldByName(fieldName)
	if !exists {
		return nil, ErrTargetFieldNotStructField
	}
	if fieldName == e.identifierFieldName && e.identifierFieldName != "" {
		if err := e.validateIdentifierValues(map[string]string{fieldName: value}); err != nil {
			return nil, err
		}
	}
	search, ok := field.Value.Label()
	if !ok {
		return nil, ErrTargetFieldNotStructField
	}
	replacement, err := metadata.FormatFieldValue(search, field.Value, value)
	if err != nil {
		return nil, err
	}

	entryRange, err := e.originalSourceRange()
	if err != nil {
		return nil, err
	}
	fields, err := entrySyntaxFromWrittenSource(e.Value, e.source[entryRange.Head:entryRange.Tail])
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
	return entryReplacementPatch(e, entryRange, content)
}

// RemoveFieldValue creates a patch that omits a field from this entry's
// concrete values without materializing optional fields or changing the
// surrounding document.
func (e Entry) RemoveFieldValue(fieldName string) (Patch, error) {
	if e.Value.IncompleteKind() != cue.StructKind {
		return nil, ErrEntryNotStruct
	}
	entryRange, err := e.originalSourceRange()
	if err != nil {
		return nil, err
	}
	fields, err := entrySyntaxFromWrittenSource(e.Value, e.source[entryRange.Head:entryRange.Tail])
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
	if fieldName == e.identifierFieldName && e.identifierFieldName != "" {
		return nil, ErrEntryIdentifierImmutable
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
	return entryReplacementPatch(e, entryRange, content)
}

// MergeFieldValues creates a patch that merges field values into this entry.
func (e Entry) MergeFieldValues(values map[string]string) (Patch, error) {
	if err := e.validateIdentifierValues(values); err != nil {
		return nil, err
	}
	patch, err := e.MergeFieldValuesAndRemove(values, nil)
	if err != nil {
		return nil, err
	}
	return identifyEntryPatch(e, patch)
}

// MergeFieldValuesAndRemove merges field values and removes named fields in one
// entry rewrite. Removal takes precedence if a field appears in both arguments.
func (e Entry) MergeFieldValuesAndRemove(values map[string]string, removals []string) (Patch, error) {
	if err := e.validateIdentifierValues(values); err != nil {
		return nil, err
	}
	if err := e.validateIdentifierRemovals(removals); err != nil {
		return nil, err
	}
	if e.Value.IncompleteKind() != cue.StructKind {
		return nil, ErrEntryNotStruct
	}

	all := make(map[string]cue.Value)
	allFields, err := e.Value.Fields(cue.All())
	if err != nil {
		return nil, err
	}
	for allFields.Next() {
		all[allFields.Selector().Unquoted()] = allFields.Value()
	}
	if len(values) == 0 && len(removals) == 0 {
		return nothingPatch{}, nil
	}
	remove := make(map[string]struct{}, len(removals))
	for _, label := range removals {
		remove[label] = struct{}{}
	}

	labels := make([]string, 0, len(values))
	for label := range values {
		if _, shouldRemove := remove[label]; !shouldRemove {
			labels = append(labels, label)
		}
	}
	sort.Strings(labels)

	formattedValues := make(map[string]string, len(labels))
	for _, label := range labels {
		value := values[label]
		if known, ok := all[label]; ok {
			value, err = metadata.FormatAccordingToAttributes(known, value)
			if err != nil {
				return nil, err
			}
		}
		formattedValues[label] = value
	}

	entryRange, err := e.originalSourceRange()
	if err != nil {
		return nil, err
	}
	fields, err := entrySyntaxFromWrittenSource(e.Value, e.source[entryRange.Head:entryRange.Tail])
	if err != nil {
		return nil, err
	}

	existing := make(map[string]*ast.Field, len(fields.Elts))
	kept := fields.Elts[:0]
	changed := false
	for _, element := range fields.Elts {
		field, ok := element.(*ast.Field)
		if !ok {
			kept = append(kept, element)
			continue
		}
		label, _, labelErr := ast.LabelName(field.Label)
		if labelErr == nil {
			if _, shouldRemove := remove[label]; shouldRemove {
				changed = true
				continue
			}
			existing[label] = field
		}
		kept = append(kept, element)
	}
	fields.Elts = kept

	for _, label := range labels {
		updated := ast.NewLit(
			token.STRING,
			literal.String.
				WithOptionalTabIndent(1).
				Quote(formattedValues[label]),
		)
		if field, ok := existing[label]; ok {
			field.Value = updated
			changed = true
			continue
		}
		fields.Elts = append(fields.Elts, &ast.Field{
			Label: ast.NewString(label),
			Value: updated,
		})
		changed = true
	}

	if !changed {
		return nothingPatch{}, nil
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
	return entryReplacementPatch(e, entryRange, content)
}
