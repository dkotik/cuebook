package cuebook

import (
	"bytes"
	"log/slog"

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
	if !validByteRange(r, source) {
		return ByteAnchor{}
	}
	content := source[r.Head:r.Tail]
	prefixStart := max(0, r.Head-64)
	suffixEnd := min(len(source), r.Tail+64)
	return ByteAnchor{
		Content:              bytes.Clone(content),
		PreceedingDuplicates: bytes.Count(source[:r.Head], content),
		Prefix:               bytes.Clone(source[prefixStart:r.Head]),
		Suffix:               bytes.Clone(source[r.Tail:suffixEnd]),
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
	Prefix               []byte
	Suffix               []byte
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

// MatchUnique locates an anchor only when its content and surrounding context
// identify exactly one range. It never falls back to a different duplicate.
func (a ByteAnchor) MatchUnique(source []byte) (ByteRange, error) {
	if a.PreceedingDuplicates < 0 {
		return ByteRange{}, ErrByteRangeNotFound
	}
	if len(a.Content) == 0 && len(a.Prefix) == 0 && len(a.Suffix) == 0 {
		if a.PreceedingDuplicates == 1 && len(source) == 0 {
			return ByteRange{}, nil
		}
		return ByteRange{}, ErrByteRangeNotFound
	}

	var match ByteRange
	found := false
	consider := func(r ByteRange) error {
		if !a.matchesContext(source, r) {
			return nil
		}
		if found {
			return ErrPatchTargetAmbiguous
		}
		match = r
		found = true
		return nil
	}

	if len(a.Content) == 0 {
		for offset := 0; offset <= len(source); offset++ {
			if err := consider(ByteRange{Head: offset, Tail: offset}); err != nil {
				return ByteRange{}, err
			}
		}
	} else {
		for searchFrom := 0; searchFrom <= len(source)-len(a.Content); {
			index := bytes.Index(source[searchFrom:], a.Content)
			if index < 0 {
				break
			}
			head := searchFrom + index
			if err := consider(ByteRange{Head: head, Tail: head + len(a.Content)}); err != nil {
				return ByteRange{}, err
			}
			searchFrom = head + 1
		}
	}
	if !found {
		return ByteRange{}, ErrByteRangeNotFound
	}
	return match, nil
}

func (a ByteAnchor) matchesContext(source []byte, r ByteRange) bool {
	if !validByteRange(r, source) || len(a.Prefix) > r.Head || len(a.Suffix) > len(source)-r.Tail {
		return false
	}
	return bytes.Equal(source[r.Head-len(a.Prefix):r.Head], a.Prefix) &&
		bytes.Equal(source[r.Tail:r.Tail+len(a.Suffix)], a.Suffix)
}

func cloneByteAnchor(anchor ByteAnchor) ByteAnchor {
	anchor.Content = bytes.Clone(anchor.Content)
	anchor.Prefix = bytes.Clone(anchor.Prefix)
	anchor.Suffix = bytes.Clone(anchor.Suffix)
	return anchor
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
