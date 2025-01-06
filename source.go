package cuebook

import (
	"bytes"
	"errors"
	"io"
	"math"

	"cuelang.org/go/cue"
)

func IsSame(a, b cue.Value) bool {
	return a.Source() == b.Source()
}

type SourcePatch struct {
	SourceByteRange
	ReplaceWith         []byte
	Original            []byte
	PrecedingDuplicates int
}

type SourcePatchResult struct {
	SourcePatch
	Book   CueBook
	Source []byte
}

func (r SourcePatchResult) WriteTo(w io.Writer) (int64, error) {
	return io.Copy(w, bytes.NewReader(r.Source))
}

func (p SourcePatch) FindOriginal(source []byte) (r SourceByteRange, err error) {
loop:
	for i := p.PrecedingDuplicates + 1; i > 0; i-- {
		switch index := bytes.Index(source, p.Original); index {
		case -1:
			// return r, errors.New("should have found")
			break loop
		default:
			r.BeginsAt = index
			r.EndsAt = index + len(p.Original)
			source = source[r.EndsAt:]
		}
	}
	if r.BeginsAt == 0 {
		return r, errors.New("failed to locate the entry designated for patch") // TODO: model error
	}
	return r, nil
}

func (p SourcePatch) Apply(source []byte) (r SourcePatchResult, err error) {
	r.Book, err = New(source)
	if err != nil {
		return r, err
	}
	latestRange, err := p.FindOriginal(source)
	if err != nil {
		return r, err
	}
	r.SourcePatch = p
	b := &bytes.Buffer{}
	b.Grow(len(source) + len(p.ReplaceWith) - len(p.Original))
	_, _ = io.Copy(b, bytes.NewReader(source[:latestRange.BeginsAt]))
	_, _ = io.Copy(b, bytes.NewReader(p.ReplaceWith))
	_, _ = io.Copy(b, bytes.NewReader(source[latestRange.EndsAt:]))
	r.Source = b.Bytes()
	r.Book, err = New(r.Source) // parse the entire book after the patch
	if err != nil {
		return r, err
	}
	return r, nil
}

type SourceByteRange struct {
	BeginsAt int
	EndsAt   int
}

func (r SourceByteRange) IsValid() bool {
	return r.BeginsAt < r.EndsAt
}

func (r SourceByteRange) Len() int {
	return r.EndsAt - r.BeginsAt
}

// TODO: add tests
func (r SourceByteRange) IsTouching(other SourceByteRange) bool {
	if r.BeginsAt < other.BeginsAt && r.EndsAt < other.BeginsAt {
		return false // r is completely before other
	}
	if r.BeginsAt > other.EndsAt && r.EndsAt > other.EndsAt {
		return false // r is completely after other
	}
	return true
}

func getTabulationRecommendationFromTail(b []byte) int {
	spaces := 0
	tabs := 0

loop:
	for i := len(b) - 1; i >= 0; i-- {
		switch b[i] {
		case ' ':
			spaces++
		case '\t':
			tabs++
		default:
			break loop
		}
	}
	if spaces > 1 {
		tabs += int(math.Ceil(float64(spaces) / 4.0))
	}
	return max(tabs, 2)
}
