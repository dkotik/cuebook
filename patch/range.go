package patch

import (
	"bytes"

	"cuelang.org/go/cue"
)

type ByteRange struct {
	Head int
	Tail int
}

func NewByteRange(v cue.Value) (byteRange ByteRange, err error) {
	_, expressions := v.Expr()
	for _, expression := range expressions {
		if expression.IsConcrete() {
			if source := expression.Source(); source != nil {
				// found first concrete data definition
				// with present source pointer
				byteRange.Head = source.Pos().Offset()
				byteRange.Tail = source.End().Offset()
				return byteRange, nil
			}
		}
	}
	return byteRange, ErrByteRangeNotFound
}

func (r ByteRange) Anchor(source []byte) ByteAnchor {
	content := source[r.Head:r.Tail]
	return ByteAnchor{
		Content:              content,
		PreceedingDuplicates: bytes.Count(source[:r.Head], content),
		// ApproximateHead:      r.Head,
	}
}

func (r ByteRange) PreceedingEntryAnchor(source []byte) ByteAnchor {
	previousListHead := bytes.LastIndex(source[:r.Head], []byte("{"))
	if previousListHead == -1 {
		// there are no preceeding structs
		previousListHead = max(0, r.Head-1000)
	}
	return ByteRange{
		Head: previousListHead,
		Tail: r.Head,
	}.Anchor(source)
}

type ByteAnchor struct {
	Content              []byte
	PreceedingDuplicates int
	// ApproximateHead      int
}

// Match locates the [ByteRange] that
// correponds to [ByteAnchor.Content]
// by finding the exact bytes and skipping
// likely identical duplicates from the
// beginning of the Cue source.
func (a ByteAnchor) Match(source []byte) (r ByteRange, err error) {
	index, length := -1, len(a.Content)
	for i := range a.PreceedingDuplicates + 1 {
		index = bytes.Index(source[r.Tail:], a.Content)
		if index == -1 {
			if i == 0 {
				return r, ErrByteRangeNotFound
			}
			break
		}
		r.Head = r.Tail + index
		r.Tail = r.Head + length
	}
	return r, nil
}
