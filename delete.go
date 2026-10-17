package cuebook

import (
	"unicode"

	"cuelang.org/go/cue"
)

type deletePatch struct {
	Preceeding ByteAnchor
	Target     ByteAnchor
}

func (p deletePatch) Difference() ByteAnchor {
	return p.Preceeding
}

func (p deletePatch) ApplyToCueSource(source []byte) ([]byte, error) {
	r, err := p.Target.Match(source)
	if err != nil {
		return nil, err
	}
	return spliceRange(source, r, nil)
}

func (p deletePatch) Invert() Patch {
	return insertAfter(p)
}

func DeleteFromStructList(source []byte, value cue.Value) (Patch, error) {
	r, err := NewByteRange(value)
	if err != nil {
		return nil, err
	}
	if !validByteRange(r, source) {
		return nil, ErrByteRangeNotFound
	}

	// Extend the removed range through the following comma and line break.
	limit := min(len(source), r.Tail+100)
	for i, c := range source[r.Tail:limit] {
		if unicode.IsSpace(rune(c)) {
			continue
		}
		if c == ',' {
			r.Tail += i + 1
			for i, c = range source[r.Tail:limit] {
				if unicode.IsSpace(rune(c)) {
					if c != '\n' {
						continue
					}
				}
				r.Tail += i
				break
			}
		}
		break
	}

	// Include indentation before the entry, and one preceding newline if present.
	for i := r.Head - 1; i > max(0, r.Head-1000); i-- {
		c := rune(source[i])
		if !unicode.IsSpace(c) {
			break
		}
		r.Head--
		if c == '\n' {
			break
		}
	}

	preceeding := r.PreceedingEntryAnchor(source)
	target, err := anchorForRange(r, source)
	if err != nil {
		return nil, err
	}
	return deletePatch{
		Preceeding: preceeding,
		Target:     target,
	}, nil
}
