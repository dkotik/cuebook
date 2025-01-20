package cuebook

import (
	"bytes"
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/literal"
)

type Field struct {
	Parent Entry
	Name   string
	Value  cue.Value
}

// FieldDefinition lists are composed into [SourcePatch]s for new [Entry]s.
type FieldDefinition struct {
	Name         string
	EncodedValue string
}

func (f Field) String() string {
	switch k := f.Value.Kind(); k {
	case cue.BoolKind:
		value, err := f.Value.Bool()
		if err != nil || !value {
			return "X"
		}
		return booleanTrue
	case cue.IntKind, cue.FloatKind, cue.NumberKind:
		b := &bytes.Buffer{}
		_, _ = fmt.Fprintf(b, "%v", f.Value)
		return b.String()
	case cue.StringKind:
		value, _ := f.Value.String()
		return value
	case cue.BytesKind, cue.ListKind, cue.StructKind:
		value, _ := f.Value.MarshalJSON()
		return string(value)
	case cue.NullKind, cue.BottomKind:
		fallthrough
	default:
		return ""
	}
}

func (f Field) MarshallText() ([]byte, error) {
	switch k := f.Value.Kind(); k {
	case cue.BoolKind:
		value, err := f.Value.Bool()
		if err != nil || !value {
			return nil, err
		}
		return []byte(booleanTrue), nil
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
