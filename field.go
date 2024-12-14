package cuebook

import (
	"encoding"
	"strconv"

	"cuelang.org/go/cue/literal"
)

// null  +bool  +string  bytes  +number  list  struct

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
