package field

import (
	"bytes"
	"fmt"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/literal"
	"github.com/dkotik/cuebook/metadata"
)

type textMarshaler interface {
	MarshallText(cue.Value) ([]byte, error)
}

// Field represents a named CUE field and its preceding documentation comments.
type Field struct {
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
	return Field{
		Name:        name,
		Description: description.String(),
		Value:       value,
	}
}

func marshalerFor(value cue.Value) textMarshaler {
	kind := value.IncompleteKind()
	if kind&cue.StringKind != 0 {
		if metadata.IsMultiLine(value) {
			return multiline{}
		}
		if isPassword(value) {
			return password{}
		}
		return stringValue{}
	}
	switch {
	case kind&cue.BoolKind != 0:
		return boolean{}
	case kind == cue.IntKind:
		return integer{}
	case kind&cue.NumberKind != 0:
		return float{}
	default:
		return nil
	}
}

func (f Field) WithValue(value string) (*ast.Field, error) {
	return metadata.FormatFieldValue(f.Name, f.Value, value)
}

func (f Field) Default() (string, bool) {
	return metadata.GetDefaultValue(f.Value)
}

func (f Field) String() string {
	return metadata.ValueToString(f.Value)
}

func (f Field) MarshallText() ([]byte, error) {
	if marshaler := marshalerFor(f.Value); marshaler != nil {
		return marshaler.MarshallText(f.Value)
	}
	switch kind := f.Value.Kind(); kind {
	case cue.BytesKind, cue.ListKind, cue.StructKind:
		return f.Value.MarshalJSON()
	case cue.NullKind:
		return []byte(`null`), nil
	case cue.BottomKind:
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown data type: %s", kind)
	}
}

// MarshalText implements encoding.TextMarshaler.
func (f Field) MarshalText() ([]byte, error) { return f.MarshallText() }

func marshalNumber(value cue.Value) ([]byte, error) {
	b := &bytes.Buffer{}
	_, err := fmt.Fprintf(b, "%v", value)
	return b.Bytes(), err
}

func quoteString(value string) string {
	return literal.String.Quote(value)
}

func quoteMultiline(value string) string {
	return literal.String.WithOptionalTabIndent(1).Quote(value)
}
