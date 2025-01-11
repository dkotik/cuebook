package patch

import (
	"bytes"
	"io"
	"unicode"

	"cuelang.org/go/cue"
)

type delete struct {
	Preceeding ByteAnchor
	Target     ByteAnchor
}

func (p delete) ApplyToCueSource(source []byte) (result []byte, err error) {
	r, err := p.Target.Match(source)
	if err != nil {
		return nil, err
	}
	b := &bytes.Buffer{}
	_, _ = io.Copy(b, bytes.NewReader(source[:r.Head]))
	_, _ = io.Copy(b, bytes.NewReader(source[r.Tail:]))
	return b.Bytes(), nil
}

func (p delete) Invert() Patch {
	return insertAfter(p)
}

func DeleteFromStructList(source []byte, value cue.Value) (Patch, error) {
	r, err := NewByteRange(value)
	if err != nil {
		return nil, err
	}

	// extend range to list comma
	limit := min(len(source), r.Tail+100)
	for i, c := range source[r.Tail:limit] {
		if unicode.IsSpace(rune(c)) {
			continue
		}
		if c == ',' {
			r.Tail += i + 1
			for i, c = range source[r.Tail:limit] {
				if unicode.IsSpace(rune(c)) {
					if c != '\n' { // until end of line
						continue
					}
					i++ // take new line
				}
				r.Tail += i
				break
			}
		}
		break
	}

	return delete{
		Preceeding: r.PreceedingEntryAnchor(source),
		Target:     r.Anchor(source),
	}, nil
}
