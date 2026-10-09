package cuebook

import (
	"bytes"
	"fmt"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/literal"
	"cuelang.org/go/cue/token"
	"github.com/dkotik/cuebook/metadata"
)

type Field struct {
	// Parent Entry
	Name        string
	Description string
	Value       cue.Value
}

// NewField creates a field from a CUE value and its preceding documentation comments.
func NewField(name string, value cue.Value) Field {
	var description strings.Builder
	for _, comment := range value.Doc() {
		text := strings.TrimSpace(comment.Text())
		if text == "" {
			continue
		}
		if description.Len() > 0 {
			description.WriteByte('\n')
		}
		description.WriteString(text)
	}
	return Field{Name: name, Description: description.String(), Value: value}
}

func (f Field) WithValue(value string) (_ *ast.Field, err error) {
	value, err = metadata.FormatAccordingToAttributes(f.Value, value)
	if err != nil {
		return nil, fmt.Errorf("failed to format field value: %w", err)
	}
	return &ast.Field{
		Label: ast.NewString(f.Name),
		Value: ast.NewLit(
			token.STRING,
			literal.String.
				WithOptionalTabIndent(1).
				Quote(value),
		),
	}, nil
}

func (f Field) Default() (string, bool) {
	return metadata.GetDefaultValue(f.Value)
}

func (f Field) String() string {
	return metadata.ValueToString(f.Value)
}

func (f Field) MarshallText() ([]byte, error) {
	switch k := f.Value.Kind(); k {
	case cue.BoolKind:
		value, err := f.Value.Bool()
		if err != nil || !value {
			return nil, err
		}
		return []byte("true"), nil
	case cue.IntKind, cue.FloatKind, cue.NumberKind:
		b := &bytes.Buffer{}
		_, err := fmt.Fprintf(b, "%v", f.Value)
		return b.Bytes(), err // TODO: test this
	case cue.StringKind:
		value, err := f.Value.String()
		return []byte(literal.String.WithOptionalTabIndent(1).Quote(value)), err
	case cue.BytesKind, cue.ListKind, cue.StructKind:
		return f.Value.MarshalJSON()
	case cue.NullKind:
		return []byte(`null`), nil
	case cue.BottomKind:
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown data type: %s", k)
	}
}
