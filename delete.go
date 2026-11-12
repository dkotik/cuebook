package cuebook

import "unicode"

// DeleteFromStructList creates a patch that removes this entry from its list.
func (e Entry) DeleteFromStructList() (Patch, error) {
	r, err := e.originalSourceRange()
	if err != nil {
		return nil, err
	}
	return deleteByteRange(e.source, r, e.precedingSourceRange, e.hasPrecedingSourceRange)
}

func deleteByteRange(source []byte, r, precedingRange ByteRange, hasPrecedingRange bool) (Patch, error) {
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

	var difference ByteAnchor
	if hasPrecedingRange && validByteRange(precedingRange, source) {
		updated, err := spliceRange(source, r, nil)
		if err != nil {
			return nil, err
		}
		difference = precedingRange.Anchor(updated)
	}
	return replaceByteRangeWithDifference(source, r, nil, difference)
}
