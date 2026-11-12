package cuebook

import (
	"bytes"
	"unicode"
)

// DeleteFromStructList creates a patch that removes this entry from its list.
func (e Entry) DeleteFromStructList() (Patch, error) {
	r, err := e.originalSourceRange()
	if err != nil {
		return nil, err
	}
	identity, identified := e.entryIdentity()
	if !identified {
		return deleteByteRange(e.source, r, e.precedingSourceRange, e.hasPrecedingSourceRange)
	}

	book, entries, err := entriesFromSource(e.source)
	if err != nil {
		return nil, err
	}
	index, found, err := findEntryIdentity(book, entries, identity)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, ErrPatchTargetConflict
	}
	entry := entries[index]
	r, err = entry.originalSourceRange()
	if err != nil {
		return nil, err
	}

	var left, right entryLocator
	hasLeft, hasRight := index > 0, index+1 < len(entries)
	if hasLeft {
		left, err = entryLocatorFor(entries[index-1], e.source)
		if err != nil {
			return nil, err
		}
	}
	if hasRight {
		right, err = entryLocatorFor(entries[index+1], e.source)
		if err != nil {
			return nil, err
		}
	}

	change, err := deleteByteRange(e.source, r, entry.precedingSourceRange, entry.hasPrecedingSourceRange)
	if err != nil {
		return nil, err
	}
	deleted, ok := change.(replacePatch)
	if !ok {
		return nil, ErrPatchTargetConflict
	}
	updated, err := change.ApplyToCueSource(e.source)
	if err != nil {
		return nil, err
	}
	var emptyGap ByteAnchor
	if !hasLeft && !hasRight {
		deletedRange, err := deleted.Target.MatchUnique(e.source)
		if err != nil {
			return nil, err
		}
		emptyGap = (ByteRange{Head: deletedRange.Head, Tail: deletedRange.Head}).Anchor(updated)
	}

	return identifiedEntryDeletePatch{
		target:     identity,
		before:     bytes.Clone(e.source[r.Head:r.Tail]),
		removed:    bytes.Clone(deleted.Target.Content),
		left:       cloneEntryLocator(left),
		hasLeft:    hasLeft,
		right:      cloneEntryLocator(right),
		hasRight:   hasRight,
		emptyGap:   cloneByteAnchor(emptyGap),
		difference: cloneByteAnchor(deleted.Difference()),
	}, nil
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
