package cuebook

import (
	"bytes"
	"encoding"
	"errors"
	"fmt"
	"io"
	"strconv"
	"unicode"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/literal"
)

type Field struct {
	Name  string
	Value cue.Value
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
		return informationUnavailable
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

func (f Field) substituteWithValue(w io.Writer, source, quotedValue []byte) error {
	at := GetByteSpanInSource(f.Value)
	if !at.IsValid() {
		return fmt.Errorf("unable to determine field bounds") // TODO: implement error
	}
	labelEnds := at.BeginsAt + len(f.Name)
	if f.Name+":" != string(source[at.BeginsAt:labelEnds+1]) {
		return fmt.Errorf("field name mismatch: %q vs %q", f.Name, string(source[at.BeginsAt:labelEnds]))
	}

	var err error
	for labelEnds++; labelEnds < at.EndsAt; labelEnds++ {
		if !unicode.IsSpace(rune(source[labelEnds])) {
			labelEnds--
			// quotedValue, err := f.MarshallText()
			// if err != nil {
			// 	return err
			// }
			if _, err = io.Copy(w, bytes.NewReader(source[:labelEnds])); err != nil {
				return err
			}
			if _, err = w.Write([]byte(" ")); err != nil {
				return err
			}
			if bytes.HasPrefix(quotedValue, []byte("\"\"\"\n")) {
				lines := bytes.Split(quotedValue, []byte("\n"))
				if _, err = w.Write(lines[0]); err != nil {
					return err
				}
				tabs := []byte("\n")
				for range getTabulationRecommendationFromTail(source[:at.BeginsAt]) {
					tabs = append(tabs, '\t')
				}
				for _, line := range lines[1:] {
					if _, err = w.Write(tabs); err != nil {
						return err
					}
					if _, err = w.Write(line); err != nil {
						return err
					}
				}
			} else {
				if _, err = w.Write(quotedValue); err != nil {
					return err
				}
			}
			_, err = io.Copy(w, bytes.NewReader(source[at.EndsAt:]))
			return err
		}
	}
	return errors.New("unable to find label end") // TODO: implement error
}

// DEPRECATED
type FieldType interface {
	GetName() string
	encoding.TextMarshaler
	// WithValue(string) (Field, error)
	String() string
}

var (
	_ FieldType = (*stringField)(nil)
	_ FieldType = (*integerField)(nil)
	_ FieldType = (*floatField)(nil)
	_ FieldType = (*booleanField)(nil)
)

type stringField struct {
	Name  string
	Value string
}

func (s stringField) GetName() string {
	return s.Name
}

func (s stringField) String() string {
	return s.Value
}

func (s stringField) MarshalText() (text []byte, err error) {
	return []byte(literal.String.WithOptionalTabIndent(2).Quote(s.Value)), nil
}

type integerField struct {
	Name  string
	Value int64
}

func (i integerField) GetName() string {
	return i.Name
}

func (i integerField) String() string {
	return strconv.Itoa(int(i.Value))
}

func (i integerField) MarshalText() (text []byte, err error) {
	return []byte(i.String()), nil
}

type floatField struct {
	Name  string
	Value float64
}

func (f floatField) GetName() string {
	return f.Name
}

func (f floatField) String() string {
	return strconv.Itoa(int(f.Value))
}

func (f floatField) MarshalText() (text []byte, err error) {
	return []byte(f.String()), nil
}

type booleanField struct {
	Name  string
	Value bool
}

func (b booleanField) GetName() string {
	return b.Name
}

func (b booleanField) String() string {
	if b.Value {
		return "true"
	}
	return "false"
}

func (b booleanField) MarshalText() (text []byte, err error) {
	return []byte(b.String()), nil
}
