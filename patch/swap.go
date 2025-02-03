package patch

import (
	"bytes"
	"io"

	"cuelang.org/go/cue"
)

type swapPatch struct {
	Earlier                ByteAnchor
	Later                  ByteAnchor
	EarlierIsTarget        bool
	EarlierDuplicatesInGap int
	LaterDuplicatesInGap   int
}

func (p swapPatch) Difference() ByteAnchor {
	if p.EarlierIsTarget {
		return p.Earlier
	}
	return p.Later
}

func (p swapPatch) ApplyToCueSource(source []byte) (result []byte, err error) {
	earlier, err := p.Earlier.Match(source)
	if err != nil {
		return nil, err
	}
	p.Later.PreceedingDuplicates += p.LaterDuplicatesInGap
	later, err := p.Later.Match(source)
	if err != nil {
		return nil, err
	}
	b := &bytes.Buffer{}
	b.Grow(len(source))
	_, _ = io.Copy(b, bytes.NewReader(source[:earlier.Head]))
	_, _ = io.Copy(b, bytes.NewReader(source[later.Head:later.Tail]))
	_, _ = io.Copy(b, bytes.NewReader(source[earlier.Tail:later.Head]))
	_, _ = io.Copy(b, bytes.NewReader(source[earlier.Head:earlier.Tail]))
	_, _ = io.Copy(b, bytes.NewReader(source[later.Tail:]))
	return b.Bytes(), nil
}

func (p swapPatch) Invert() Patch {
	return swapPatch{
		Earlier:                p.Later,
		Later:                  p.Earlier,
		EarlierIsTarget:        !p.EarlierIsTarget,
		EarlierDuplicatesInGap: p.LaterDuplicatesInGap,
		LaterDuplicatesInGap:   p.EarlierDuplicatesInGap,
	}
}

func SwapEntries(source []byte, a, b cue.Value) (Patch, error) {
	aRange, err := NewByteRange(a)
	if err != nil {
		return nil, err
	}
	bRange, err := NewByteRange(b)
	if err != nil {
		return nil, err
	}
	if bytes.Equal(source[aRange.Head:aRange.Tail], source[bRange.Head:bRange.Tail]) {
		if aRange.Head > bRange.Head {
			return nothingPatch{
				A: bRange.Anchor(source),
				B: aRange.Anchor(source),
			}, nil
		}
		return nothingPatch{
			A: aRange.Anchor(source),
			B: bRange.Anchor(source),
		}, nil
	}
	p := swapPatch{}
	if aRange.Head > bRange.Head {
		aRange, bRange = bRange, aRange
	} else {
		p.EarlierIsTarget = true
	}
	if aRange.Tail > bRange.Head {
		return nil, ErrByteRangesOverlap
	}
	p.Earlier = aRange.Anchor(source)
	p.Later = bRange.Anchor(source)
	if aRange.Tail < bRange.Head {
		gap := source[aRange.Tail:bRange.Head]
		p.EarlierDuplicatesInGap = bytes.Count(gap, p.Earlier.Content)
		p.LaterDuplicatesInGap = bytes.Count(gap, p.Later.Content)
		p.Later.PreceedingDuplicates -= p.LaterDuplicatesInGap
	}
	return p, nil
}
