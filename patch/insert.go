package patch

import (
	"bytes"
	"errors"
	"io"
	"unicode"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/format"
)

type insertAfter struct {
	Preceeding ByteAnchor
	Target     ByteAnchor
}

func (p insertAfter) ApplyToCueSource(source []byte) (result []byte, err error) {
	r, err := p.Preceeding.Match(source)
	if err != nil {
		return nil, err
	}
	b := &bytes.Buffer{}
	_, _ = io.Copy(b, bytes.NewReader(source[:r.Tail]))
	_, _ = io.Copy(b, bytes.NewReader(p.Target.Content))
	_, _ = io.Copy(b, bytes.NewReader(source[r.Tail:]))
	return b.Bytes(), nil
}

func (p insertAfter) Invert() Patch {
	return delete(p)
}

func AppendToStructList(source []byte, value cue.Value) (Patch, error) {
	content, err := format.Node(
		value.Syntax(),
		format.Simplify(),
		format.IndentPrefix(1),
		format.UseSpaces(4),
	)
	if err != nil {
		return nil, err
	}
	if len(source) < 2 {
		source = []byte("[\n]\n")
	}
	listEnd, commaFound, err := locateRootListEnd(source)
	if err != nil {
		return nil, err
	}
	if !commaFound {
		content = append([]byte(",\n    "), content...)
	} else {
		content = append([]byte("\n    "), content...)
	}
	return insertAfter{
		Preceeding: ByteRange{
			Head: max(0, listEnd-1000),
			Tail: listEnd,
		}.Anchor(source),
		Target: ByteAnchor{
			Content:              content,
			PreceedingDuplicates: bytes.Count(source[:listEnd], content),
		},
	}, nil
}

func locateRootListEnd(source []byte) (i int, commaFound bool, err error) {
	listEndFound := false

loop:
	for i := len(source) - 1; i >= min(0, len(source)-1000); i-- {
		switch c := source[i]; c {
		case '}':
			if commaFound {
				i++
			}
			return i + 1, commaFound, nil
		case ',':
			if commaFound {
				return 0, commaFound, errors.New("double comma at list end") // TODO: model
			}
			commaFound = true
		case ']':
			if listEndFound {
				return 0, commaFound, errors.New("not a list of structs") // TODO: model error
			}
			listEndFound = true
		default:
			if unicode.IsSpace(rune(c)) {
				continue
			}
			break loop
		}
	}
	return 0, commaFound, errors.New("root list end not found") // TODO: model error
}
