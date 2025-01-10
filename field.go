package cuebook

import (
	"bytes"
	"errors"
	"fmt"
	"unicode"

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

func (f Field) WithStringValue(source []byte, value string) (p SourcePatch, err error) {
	quotedValue := literal.String.WithOptionalTabIndent(0).Quote(value)
	return f.newValuePatch(source, []byte(quotedValue))
}

func (f Field) newValuePatch(source, quotedValue []byte) (p SourcePatch, err error) {
	at := GetByteSpanInSource(f.Value)
	if !at.IsValid() {
		return p, fmt.Errorf("unable to determine field bounds") // TODO: implement error
	}
	labelEnds := at.BeginsAt + len(f.Name) + 1
	if f.Name+":" != string(source[at.BeginsAt:labelEnds]) {
		return p, fmt.Errorf("field name mismatch: %q vs %q", f.Name, string(source[p.SourceByteRange.BeginsAt:labelEnds-1]))
	}

	for ; labelEnds < at.EndsAt; labelEnds++ {
		if !unicode.IsSpace(rune(source[labelEnds])) {
			if bytes.HasPrefix(quotedValue, []byte("\"\"\"\n")) {
				b := &bytes.Buffer{}
				lines := bytes.Split(quotedValue, []byte("\n"))
				_, _ = b.Write(lines[0])
				tabs := []byte("\n")
				for range getTabulationRecommendationFromTail(source[:at.BeginsAt]) {
					tabs = append(tabs, '\t')
				}
				for _, line := range lines[1:] {
					_, _ = b.Write(tabs)
					_, _ = b.Write(line)
				}
				quotedValue = b.Bytes()
			}
			parentSpan := GetByteSpanInSource(f.Parent.Value)
			if !parentSpan.IsValid() {
				return p, fmt.Errorf("unable to determine parent entry bounds") // TODO: implement error
			}
			p.SourceByteRange = parentSpan
			p.ReplaceWith = bytes.Join([][]byte{
				source[parentSpan.BeginsAt:labelEnds],
				quotedValue,
				source[at.EndsAt:parentSpan.EndsAt],
			}, nil)
			p.Original = source[parentSpan.BeginsAt:parentSpan.EndsAt]
			p.PrecedingDuplicates = bytes.Count(source[:at.EndsAt], p.Original)
			return p, nil
		}
	}
	return p, errors.New("unable to find entry field label end") // TODO: implement error
}

func (f Field) RemoveFromSource(source []byte) []byte {
	at := GetByteSpanInSource(f.Value)
	if !at.IsValid() {
		return nil
	}

	return append(
		chopLeadingWhiteSpace(source[:at.BeginsAt]),
		chopFieldClosingCommaWithNewLine(source[at.EndsAt:])...,
	)
}

func chopLeadingWhiteSpace(source []byte) []byte {
	for i := len(source) - 1; i >= 0; i-- {
		if !unicode.IsSpace(rune(source[i])) {
			return source[:i+1]
		}
		if source[i] == '\n' {
			return source[:i+1]
		}
	}
	return source
}

func chopFieldClosingCommaWithNewLine(source []byte) []byte {
	for i, c := range source {
		if unicode.IsSpace(rune(c)) {
			continue
		}
		if c == ',' {
			source = source[i+1:]
		}
		break
	}
	for i, c := range source {
		if c == '\n' {
			return source[i+1:]
		}
		if !unicode.IsSpace(rune(c)) {
			break
		}
	}
	return source
}
