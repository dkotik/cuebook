package patch

import (
	"bytes"
	"io"

	"cuelang.org/go/cue"
)

type replacePatch struct {
	Target      ByteAnchor
	Replacement ByteAnchor
}

func (p replacePatch) Difference() ByteAnchor {
	return p.Replacement
}

func (p replacePatch) ApplyToCueSource(source []byte) (result []byte, err error) {
	r, err := p.Target.Match(source)
	if err != nil {
		return nil, err
	}
	b := &bytes.Buffer{}
	_, _ = io.Copy(b, bytes.NewReader(source[:r.Head]))
	_, _ = io.Copy(b, bytes.NewReader(p.Replacement.Content))
	_, _ = io.Copy(b, bytes.NewReader(source[r.Tail:]))
	return b.Bytes(), nil
}

func (p replacePatch) Invert() Patch {
	return replacePatch{
		Target:      p.Replacement,
		Replacement: p.Target,
	}
}

func ReplaceStructListEntry(source []byte, value cue.Value, b []byte) (Patch, error) {
	r, err := NewByteRange(value)
	if err != nil {
		return nil, err
	}
	return replacePatch{
		Target: r.Anchor(source),
		Replacement: ByteAnchor{
			Content:              b,
			PreceedingDuplicates: bytes.Count(source[:r.Head], b),
		},
	}, nil
}
