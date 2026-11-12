package cuebook

import (
	"bytes"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/literal"
	"cuelang.org/go/cue/parser"
	"cuelang.org/go/cue/token"
)

// FieldKind is the type of a field added with [AddField].
type FieldKind string

const (
	FieldKindString    FieldKind = "string"
	FieldKindMultiLine FieldKind = "multiline"
	FieldKindNumber    FieldKind = "number"
	FieldKindBoolean   FieldKind = "boolean"
)

// FieldKinds lists the kinds accepted by [AddField] in presentation order.
func FieldKinds() []FieldKind {
	return []FieldKind{FieldKindString, FieldKindMultiLine, FieldKindNumber, FieldKindBoolean}
}

// Label is the human-readable name of the kind.
func (k FieldKind) Label() string {
	switch k {
	case FieldKindString:
		return "String"
	case FieldKindMultiLine:
		return "Multiline string"
	case FieldKindNumber:
		return "Number"
	case FieldKindBoolean:
		return "Boolean"
	default:
		return string(k)
	}
}

// ValidateValue reports whether [AddField] accepts the value for a field of
// this kind. It returns [ErrUnknownFieldKind] or [ErrInvalidFieldValue].
func (k FieldKind) ValidateValue(value string) error {
	_, err := newFieldDeclaration("value", k, value)
	return err
}

// AddField appends a new field to the end of this entry. An empty value
// becomes the zero value of the kind: an empty string, 0, or false. A
// multiline string is marked with @cuebook(multiline) so that editors keep
// treating it as multiline.
//
// The entry is rewritten from its source, so its comments, formatting within
// fields, and attributes are kept. The patch does not check whether the entry's
// schema accepts the field: validate the patched source with [New] or wrap the
// patch with [Validated].
func (e Entry) AddField(name string, kind FieldKind, value string) (Patch, error) {
	if strings.TrimSpace(name) == "" {
		return nil, ErrFieldNameEmpty
	}
	if e.Value.IncompleteKind() != cue.StructKind {
		return nil, ErrEntryNotStruct
	}
	exists, err := hasField(e.Value, name)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrFieldExists
	}
	field, err := newFieldDeclaration(name, kind, value)
	if err != nil {
		return nil, err
	}
	entryRange, err := e.originalSourceRange()
	if err != nil {
		return nil, err
	}
	replacement, err := addFieldAtRange(e.source, entryRange, name, kind, field)
	if err != nil {
		return nil, err
	}
	return entryReplacementPatch(e, entryRange, replacement)
}

func addFieldAtRange(source []byte, entryRange ByteRange, name string, kind FieldKind, field *ast.Field) ([]byte, error) {
	if !validByteRange(entryRange, source) {
		return nil, ErrByteRangeNotFound
	}
	written := source[entryRange.Head:entryRange.Tail]
	expression, err := parser.ParseExpr("", written, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	fields, ok := expression.(*ast.StructLit)
	if !ok {
		return nil, ErrEntryNotStruct
	}
	for _, element := range fields.Elts {
		writtenField, ok := element.(*ast.Field)
		if !ok {
			continue
		}
		label, _, err := ast.LabelName(writtenField.Label)
		if err == nil && label == name {
			return nil, ErrFieldExists
		}
	}
	if !bytes.ContainsRune(written, '\n') {
		if kind == FieldKindMultiLine {
			for _, element := range fields.Elts {
				ast.SetRelPos(element, token.Newline)
			}
			fields.Rbrace = fields.Rbrace.WithRel(token.Newline)
		} else {
			ast.SetRelPos(field, token.Blank)
		}
	}
	fields.Elts = append(fields.Elts, field)
	content, err := format.Node(
		fields,
		format.Simplify(),
		format.IndentPrefix(1),
		format.UseSpaces(4),
	)
	if err != nil {
		return nil, err
	}
	return content, nil
}

// hasField includes optional fields from the schema, because adding one of
// those is editing an existing field rather than adding a new one.
func hasField(entry cue.Value, name string) (bool, error) {
	fields, err := entry.Fields(cue.Optional(true))
	if err != nil {
		return false, err
	}
	for fields.Next() {
		if fields.Selector().Unquoted() == name {
			return true, nil
		}
	}
	return false, nil
}

func newFieldDeclaration(name string, kind FieldKind, value string) (*ast.Field, error) {
	field := &ast.Field{Label: newLabel(name)}
	switch kind {
	case FieldKindString:
		field.Value = ast.NewString(value)
	case FieldKindMultiLine:
		// Browsers submit textarea lines separated by CRLF.
		value = strings.ReplaceAll(value, "\r\n", "\n")
		field.Value = ast.NewLit(token.STRING, literal.String.WithOptionalTabIndent(1).Quote(value))
		field.Attrs = []*ast.Attribute{{Text: "@cuebook(multiline)"}}
	case FieldKindNumber:
		number, err := newNumberLiteral(value)
		if err != nil {
			return nil, err
		}
		field.Value = number
	case FieldKindBoolean:
		switch strings.TrimSpace(value) {
		case "", "false":
			field.Value = ast.NewBool(false)
		case "true":
			field.Value = ast.NewBool(true)
		default:
			return nil, ErrInvalidFieldValue
		}
	default:
		return nil, ErrUnknownFieldKind
	}
	return field, nil
}

// newLabel quotes names that are not plain identifiers. Names that look like
// definitions or hidden fields stay quoted so they remain regular fields.
func newLabel(name string) ast.Label {
	if ast.IsValidIdent(name) && !strings.HasPrefix(name, "_") && !strings.HasPrefix(name, "#") {
		return ast.NewIdent(name)
	}
	return ast.NewString(name)
}

// newNumberLiteral accepts any CUE number literal, such as 42, -1.5, 1e3,
// 0x1F, or 1_000, with an optional sign.
func newNumberLiteral(value string) (ast.Expr, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return ast.NewLit(token.INT, "0"), nil
	}
	expression, err := parser.ParseExpr("", value)
	if err != nil {
		return nil, ErrInvalidFieldValue
	}
	var sign token.Token
	if unary, ok := expression.(*ast.UnaryExpr); ok && (unary.Op == token.SUB || unary.Op == token.ADD) {
		sign, expression = unary.Op, unary.X
	}
	number, ok := expression.(*ast.BasicLit)
	if !ok || (number.Kind != token.INT && number.Kind != token.FLOAT) {
		return nil, ErrInvalidFieldValue
	}
	// Fresh nodes keep positions from the parsed value out of the entry.
	result := ast.Expr(ast.NewLit(number.Kind, number.Value))
	if sign != token.ILLEGAL {
		result = &ast.UnaryExpr{Op: sign, X: result}
	}
	return result, nil
}
