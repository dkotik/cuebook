package metadata

import (
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/literal"
	"cuelang.org/go/cue/token"
)

// FormatFieldValue converts an edited field value into CUE syntax, applying
// field metadata transforms before quoting the value.
func FormatFieldValue(name string, field cue.Value, input string) (*ast.Field, error) {
	value, err := FormatAccordingToAttributes(field, input)
	if err != nil {
		return nil, fmt.Errorf("failed to format field value: %w", err)
	}

	if field.IncompleteKind() == cue.BoolKind && !isPasswordField(field) {
		var kind token.Token
		switch value {
		case "true":
			kind = token.TRUE
		case "false":
			kind = token.FALSE
		default:
			return nil, fmt.Errorf("invalid boolean field value: %q", value)
		}
		return &ast.Field{Label: ast.NewString(name), Value: ast.NewLit(kind, value)}, nil
	}

	quoted := literal.String.Quote(value)
	if IsMultiLine(field) {
		quoted = literal.String.WithOptionalTabIndent(1).Quote(value)
	}
	return &ast.Field{
		Label: ast.NewString(name),
		Value: ast.NewLit(token.STRING, quoted),
	}, nil
}

func isPasswordField(field cue.Value) bool {
	_, ok := GetFieldAttributes(field, "cuebook").GetFirstOf("argon2id")
	return ok
}
