package patch

import (
	"bytes"
	"io"

	"cuelang.org/go/cue"
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
	// spew.Dump(string(source[r.Head:r.Tail]))
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
	r, err := NewByteRange(value)
	if err != nil {
		return nil, err
	}
	return insertAfter{
		Preceeding: r.PreceedingEntryAnchor(source),
		Target:     r.Anchor(source),
	}, nil
}
