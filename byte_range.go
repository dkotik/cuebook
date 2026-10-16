package cuebook

import (
	"bytes"
	"errors"
	"log/slog"

	"cuelang.org/go/cue"
)

// ErrByteRangeNotFound is returned when a byte range cannot be located.
var ErrByteRangeNotFound = errors.New("byte range not found")

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
	if !validByteRange(r, source) {
		return ByteAnchor{}
	}
	content := source[r.Head:r.Tail]
	return ByteAnchor{
		Content:              content,
		PreceedingDuplicates: bytes.Count(source[:r.Head], content),
		// ApproximateHead:      r.Head,
	}
}

func (r ByteRange) PreceedingEntryAnchor(source []byte) ByteAnchor {
	if !validByteRange(r, source) {
		return ByteAnchor{}
	}
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

// Match locates the [ByteRange] that corresponds to [ByteAnchor.Content] by
// skipping likely identical duplicates from the beginning of the CUE source.
// If earlier duplicates have disappeared since the anchor was created, it
// returns the last remaining match rather than rejecting an otherwise usable
// anchor.
func (a ByteAnchor) Match(source []byte) (r ByteRange, err error) {
	if a.PreceedingDuplicates < 0 {
		return r, ErrByteRangeNotFound
	}
	length := len(a.Content)
	if length == 0 {
		offset := a.PreceedingDuplicates - 1
		if offset < 0 || offset > len(source) {
			return r, ErrByteRangeNotFound
		}
		return ByteRange{Head: offset, Tail: offset}, nil
	}

	searchFrom := 0
	for i := 0; i <= a.PreceedingDuplicates; i++ {
		index := bytes.Index(source[searchFrom:], a.Content)
		if index < 0 {
			if i == 0 {
				return ByteRange{}, ErrByteRangeNotFound
			}
			break
		}
		r.Head = searchFrom + index
		r.Tail = r.Head + length
		searchFrom = r.Tail
	}
	return r, nil
}

func (a ByteAnchor) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("content", string(a.Content)),
		slog.Int("preceedingDuplicates", a.PreceedingDuplicates),
	)
}

func validByteRange(r ByteRange, source []byte) bool {
	return r.Head >= 0 && r.Head <= r.Tail && r.Tail <= len(source)
}

func anchorForRange(r ByteRange, source []byte) (ByteAnchor, error) {
	if !validByteRange(r, source) {
		return ByteAnchor{}, ErrByteRangeNotFound
	}
	return r.Anchor(source), nil
}

func spliceRange(source []byte, r ByteRange, replacement []byte) ([]byte, error) {
	if !validByteRange(r, source) {
		return nil, ErrByteRangeNotFound
	}
	result := make([]byte, 0, len(source)-(r.Tail-r.Head)+len(replacement))
	result = append(result, source[:r.Head]...)
	result = append(result, replacement...)
	result = append(result, source[r.Tail:]...)
	return result, nil
}

func spliceAt(source []byte, at int, inserted []byte) ([]byte, error) {
	if at < 0 || at > len(source) {
		return nil, ErrByteRangeNotFound
	}
	return spliceRange(source, ByteRange{Head: at, Tail: at}, inserted)
}
