package cuebook

import "bytes"

type entryIdentity struct {
	fieldName string
	value     string
}

func (identity entryIdentity) valid() bool {
	return identity.fieldName != "" && identity.value != ""
}

func (e Entry) entryIdentity() (entryIdentity, bool) {
	identity := entryIdentity{fieldName: e.identifierFieldName, value: e.UUID}
	return identity, identity.valid()
}

func resolveEntryIdentity(source []byte, identity entryIdentity) (Entry, error) {
	if !identity.valid() {
		return Entry{}, ErrPatchTargetConflict
	}
	book, err := New(source)
	if err != nil {
		return Entry{}, err
	}
	if book.entryUniqueIdentifierFieldName != identity.fieldName {
		return Entry{}, ErrPatchTargetConflict
	}

	var match Entry
	found := false
	for entry, err := range book.EachEntry() {
		if err != nil {
			return Entry{}, err
		}
		if entry.identifierFieldName != identity.fieldName || entry.UUID != identity.value {
			continue
		}
		if found {
			return Entry{}, ErrPatchTargetAmbiguous
		}
		match = entry
		found = true
	}
	if !found {
		return Entry{}, ErrPatchTargetConflict
	}
	return match, nil
}

type identifiedEntryReplacePatch struct {
	target entryIdentity
	before []byte
	after  []byte
}

func (p identifiedEntryReplacePatch) ApplyToCueSource(source []byte) ([]byte, error) {
	entry, err := resolveEntryIdentity(source, p.target)
	if err != nil {
		return nil, err
	}
	rangeInSource, err := entry.originalSourceRange()
	if err != nil {
		return nil, err
	}
	current := source[rangeInSource.Head:rangeInSource.Tail]
	if bytes.Equal(current, p.after) {
		return source, nil
	}
	if !bytes.Equal(current, p.before) {
		return nil, ErrPatchTargetConflict
	}
	return spliceRange(source, rangeInSource, p.after)
}

func (p identifiedEntryReplacePatch) Difference() ByteAnchor {
	return ByteAnchor{Content: []byte(p.target.value)}
}

func (p identifiedEntryReplacePatch) Invert() Patch {
	return identifiedEntryReplacePatch{
		target: p.target,
		before: bytes.Clone(p.after),
		after:  bytes.Clone(p.before),
	}
}

func (p identifiedEntryReplacePatch) patchTargetIdentity() (entryIdentity, bool) {
	return p.target, p.target.valid()
}

func entryReplacementPatch(e Entry, r ByteRange, replacement []byte) (Patch, error) {
	identity, identified := e.entryIdentity()
	if !identified {
		return replaceByteRange(e.source, r, replacement)
	}
	if !validByteRange(r, e.source) {
		return nil, ErrByteRangeNotFound
	}
	return identifiedEntryReplacePatch{
		target: identity,
		before: bytes.Clone(e.source[r.Head:r.Tail]),
		after:  bytes.Clone(replacement),
	}, nil
}

func identifyEntryPatch(e Entry, patch Patch) (Patch, error) {
	identity, identified := e.entryIdentity()
	if !identified {
		return patch, nil
	}
	switch patch := patch.(type) {
	case replacePatch:
		beforeRange, err := e.originalSourceRange()
		if err != nil {
			return nil, err
		}
		updated, err := patch.ApplyToCueSource(e.source)
		if err != nil {
			return nil, err
		}
		updatedEntry, err := resolveEntryIdentity(updated, identity)
		if err != nil {
			return nil, err
		}
		afterRange, err := updatedEntry.originalSourceRange()
		if err != nil {
			return nil, err
		}
		return identifiedEntryReplacePatch{
			target: identity,
			before: bytes.Clone(e.source[beforeRange.Head:beforeRange.Tail]),
			after:  bytes.Clone(updated[afterRange.Head:afterRange.Tail]),
		}, nil
	case identifiedEntryReplacePatch:
		return patch, nil
	default:
		return patch, nil
	}
}

func (e Entry) validateIdentifierValues(values map[string]string) error {
	name := e.identifierFieldName
	value, changesIdentifier := values[name]
	if name == "" || !changesIdentifier {
		return nil
	}
	field, exists := e.GetFieldByName(name)
	if !exists {
		return ErrEntryIdentifierImmutable
	}
	current, err := field.Value.String()
	if err != nil || value != current {
		return ErrEntryIdentifierImmutable
	}
	return nil
}

func (e Entry) validateIdentifierRemovals(removals []string) error {
	if e.identifierFieldName == "" {
		return nil
	}
	for _, name := range removals {
		if name == e.identifierFieldName {
			return ErrEntryIdentifierImmutable
		}
	}
	return nil
}

type entryLocator struct {
	identity entryIdentity
	content  []byte
}

func entryLocatorFor(entry Entry, source []byte) (entryLocator, error) {
	if identity, ok := entry.entryIdentity(); ok {
		return entryLocator{identity: identity}, nil
	}
	r, err := entry.originalSourceRange()
	if err != nil {
		return entryLocator{}, err
	}
	return entryLocator{content: bytes.Clone(source[r.Head:r.Tail])}, nil
}

func entriesFromSource(source []byte) (Book, []Entry, error) {
	book, err := New(source)
	if err != nil {
		return Book{}, nil, err
	}
	var entries []Entry
	for entry, err := range book.EachEntry() {
		if err != nil {
			return Book{}, nil, err
		}
		entries = append(entries, entry)
	}
	return book, entries, nil
}

func findEntryIdentity(book Book, entries []Entry, identity entryIdentity) (int, bool, error) {
	if !identity.valid() || book.entryUniqueIdentifierFieldName != identity.fieldName {
		return -1, false, ErrPatchTargetConflict
	}
	found := -1
	for index, entry := range entries {
		if entry.identifierFieldName != identity.fieldName || entry.UUID != identity.value {
			continue
		}
		if found >= 0 {
			return -1, false, ErrPatchTargetAmbiguous
		}
		found = index
	}
	return found, found >= 0, nil
}

func findEntryLocator(entries []Entry, source []byte, locator entryLocator) (int, error) {
	if !locator.identity.valid() && len(locator.content) == 0 {
		return -1, ErrPatchTargetConflict
	}
	found := -1
	for index, entry := range entries {
		matches := false
		if locator.identity.valid() {
			matches = entry.identifierFieldName == locator.identity.fieldName && entry.UUID == locator.identity.value
		} else if r, err := entry.originalSourceRange(); err == nil {
			matches = bytes.Equal(source[r.Head:r.Tail], locator.content)
		}
		if !matches {
			continue
		}
		if found >= 0 {
			return -1, ErrPatchTargetAmbiguous
		}
		found = index
	}
	if found < 0 {
		return -1, ErrPatchTargetConflict
	}
	return found, nil
}

func entryHasExpectedNeighbors(entries []Entry, source []byte, index int, left entryLocator, hasLeft bool, right entryLocator, hasRight bool) error {
	if index < 0 || index >= len(entries) {
		return ErrPatchTargetConflict
	}
	if hasLeft {
		leftIndex, err := findEntryLocator(entries, source, left)
		if err != nil {
			return err
		}
		if leftIndex != index-1 {
			return ErrPatchTargetConflict
		}
	} else if index != 0 {
		return ErrPatchTargetConflict
	}
	if hasRight {
		rightIndex, err := findEntryLocator(entries, source, right)
		if err != nil {
			return err
		}
		if rightIndex != index+1 {
			return ErrPatchTargetConflict
		}
	} else if index != len(entries)-1 {
		return ErrPatchTargetConflict
	}
	return nil
}

func locateEntryGap(book Book, entries []Entry, source []byte, left entryLocator, hasLeft bool, right entryLocator, hasRight bool, emptyGap ByteAnchor) (int, error) {
	leftIndex, rightIndex := -1, -1
	if hasLeft {
		var err error
		leftIndex, err = findEntryLocator(entries, source, left)
		if err != nil {
			return 0, err
		}
	}
	if hasRight {
		var err error
		rightIndex, err = findEntryLocator(entries, source, right)
		if err != nil {
			return 0, err
		}
	}

	switch {
	case hasLeft && hasRight:
		if rightIndex != leftIndex+1 {
			return 0, ErrPatchTargetConflict
		}
		leftRange, err := entries[leftIndex].originalSourceRange()
		if err != nil {
			return 0, err
		}
		return nextListDelimiter(source, leftRange.Tail)
	case hasLeft:
		if leftIndex != len(entries)-1 {
			return 0, ErrPatchTargetConflict
		}
		leftRange, err := entries[leftIndex].originalSourceRange()
		if err != nil {
			return 0, err
		}
		return nextListDelimiter(source, leftRange.Tail)
	case hasRight:
		if rightIndex != 0 {
			return 0, ErrPatchTargetConflict
		}
		rightRange, err := entries[rightIndex].originalSourceRange()
		if err != nil {
			return 0, err
		}
		return previousListDelimiter(source, rightRange.Head)
	default:
		if len(entries) != 0 {
			return 0, ErrPatchTargetConflict
		}
		r, err := emptyGap.MatchUnique(source)
		return r.Head, err
	}
}

func nextListDelimiter(source []byte, offset int) (int, error) {
	for offset < len(source) {
		if isCUESpace(source[offset]) {
			offset++
			continue
		}
		if offset+1 < len(source) && source[offset] == '/' && source[offset+1] == '/' {
			for offset < len(source) && source[offset] != '\n' {
				offset++
			}
			continue
		}
		if offset+1 < len(source) && source[offset] == '/' && source[offset+1] == '*' {
			end := bytes.Index(source[offset+2:], []byte("*/"))
			if end < 0 {
				return 0, ErrPatchTargetConflict
			}
			offset += end + 4
			continue
		}
		if source[offset] == ',' {
			return offset + 1, nil
		}
		return 0, ErrPatchTargetConflict
	}
	return 0, ErrPatchTargetConflict
}

func previousListDelimiter(source []byte, offset int) (int, error) {
	for offset > 0 {
		if isCUESpace(source[offset-1]) {
			offset--
			continue
		}
		lineStart := bytes.LastIndexByte(source[:offset], '\n') + 1
		if comment := bytes.Index(source[lineStart:offset], []byte("//")); comment >= 0 {
			offset = lineStart + comment
			continue
		}
		if offset >= 2 && bytes.Equal(source[offset-2:offset], []byte("*/")) {
			start := bytes.LastIndex(source[:offset-2], []byte("/*"))
			if start < 0 {
				return 0, ErrPatchTargetConflict
			}
			offset = start
			continue
		}
		delimiter := offset - 1
		if source[delimiter] == ',' || source[delimiter] == '[' {
			return delimiter + 1, nil
		}
		return 0, ErrPatchTargetConflict
	}
	return 0, ErrPatchTargetConflict
}

func isCUESpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

type identifiedEntryDeletePatch struct {
	target     entryIdentity
	before     []byte
	removed    []byte
	left       entryLocator
	hasLeft    bool
	right      entryLocator
	hasRight   bool
	emptyGap   ByteAnchor
	difference ByteAnchor
}

func (p identifiedEntryDeletePatch) ApplyToCueSource(source []byte) ([]byte, error) {
	book, entries, err := entriesFromSource(source)
	if err != nil {
		return nil, err
	}
	index, found, err := findEntryIdentity(book, entries, p.target)
	if err != nil {
		return nil, err
	}
	if !found {
		if _, err := locateEntryGap(book, entries, source, p.left, p.hasLeft, p.right, p.hasRight, p.emptyGap); err != nil {
			return nil, err
		}
		return source, nil
	}

	entry := entries[index]
	r, err := entry.originalSourceRange()
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(source[r.Head:r.Tail], p.before) {
		return nil, ErrPatchTargetConflict
	}
	change, err := deleteByteRange(source, r, entry.precedingSourceRange, entry.hasPrecedingSourceRange)
	if err != nil {
		return nil, err
	}
	return change.ApplyToCueSource(source)
}

func (p identifiedEntryDeletePatch) Difference() ByteAnchor {
	return cloneByteAnchor(p.difference)
}

func (p identifiedEntryDeletePatch) patchTargetIdentity() (entryIdentity, bool) {
	if p.hasLeft && p.left.identity.valid() {
		return p.left.identity, true
	}
	if p.hasRight && p.right.identity.valid() {
		return p.right.identity, true
	}
	return entryIdentity{}, false
}

func (p identifiedEntryDeletePatch) Invert() Patch {
	return identifiedEntryRestorePatch{
		target:   p.target,
		entry:    bytes.Clone(p.before),
		removed:  bytes.Clone(p.removed),
		left:     cloneEntryLocator(p.left),
		hasLeft:  p.hasLeft,
		right:    cloneEntryLocator(p.right),
		hasRight: p.hasRight,
		emptyGap: cloneByteAnchor(p.emptyGap),
	}
}

func cloneEntryLocator(locator entryLocator) entryLocator {
	locator.content = bytes.Clone(locator.content)
	return locator
}

type identifiedEntryRestorePatch struct {
	target   entryIdentity
	entry    []byte
	removed  []byte
	left     entryLocator
	hasLeft  bool
	right    entryLocator
	hasRight bool
	emptyGap ByteAnchor
}

func (p identifiedEntryRestorePatch) ApplyToCueSource(source []byte) ([]byte, error) {
	book, entries, err := entriesFromSource(source)
	if err != nil {
		return nil, err
	}
	index, found, err := findEntryIdentity(book, entries, p.target)
	if err != nil {
		return nil, err
	}
	if found {
		r, err := entries[index].originalSourceRange()
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(source[r.Head:r.Tail], p.entry) {
			return nil, ErrPatchTargetConflict
		}
		if err := entryHasExpectedNeighbors(entries, source, index, p.left, p.hasLeft, p.right, p.hasRight); err != nil {
			return nil, err
		}
		return source, nil
	}

	insertionPoint, err := locateEntryGap(book, entries, source, p.left, p.hasLeft, p.right, p.hasRight, p.emptyGap)
	if err != nil {
		return nil, err
	}
	return spliceAt(source, insertionPoint, p.removed)
}

func (p identifiedEntryRestorePatch) Difference() ByteAnchor {
	return ByteAnchor{Content: []byte(p.target.value)}
}

func (p identifiedEntryRestorePatch) Invert() Patch {
	return identifiedEntryDeletePatch{
		target:     p.target,
		before:     bytes.Clone(p.entry),
		removed:    bytes.Clone(p.removed),
		left:       cloneEntryLocator(p.left),
		hasLeft:    p.hasLeft,
		right:      cloneEntryLocator(p.right),
		hasRight:   p.hasRight,
		emptyGap:   cloneByteAnchor(p.emptyGap),
		difference: ByteAnchor{Content: []byte(p.target.value)},
	}
}

func (p identifiedEntryRestorePatch) patchTargetIdentity() (entryIdentity, bool) {
	return p.target, p.target.valid()
}

type identifiedEntryMovePatch struct {
	target         entryIdentity
	before         []byte
	beforeLeft     entryLocator
	hasBeforeLeft  bool
	beforeRight    entryLocator
	hasBeforeRight bool
	afterLeft      entryLocator
	hasAfterLeft   bool
	afterRight     entryLocator
	hasAfterRight  bool
}

func newIdentifiedEntryMovePatch(book Book, entries []Entry, from, to int) (Patch, error) {
	identity, identified := entries[from].entryIdentity()
	if !identified {
		return nil, ErrPatchTargetConflict
	}
	index, found, err := findEntryIdentity(book, entries, identity)
	if err != nil {
		return nil, err
	}
	if !found || index != from {
		return nil, ErrPatchTargetConflict
	}
	beforeRange, err := entries[from].originalSourceRange()
	if err != nil {
		return nil, err
	}
	beforeLeft, hasBeforeLeft, beforeRight, hasBeforeRight, err := entryNeighbors(entries, book.source, from)
	if err != nil {
		return nil, err
	}
	afterEntries := moveEntryInList(entries, from, to)
	afterLeft, hasAfterLeft, afterRight, hasAfterRight, err := entryNeighbors(afterEntries, book.source, to)
	if err != nil {
		return nil, err
	}
	return identifiedEntryMovePatch{
		target:         identity,
		before:         bytes.Clone(book.source[beforeRange.Head:beforeRange.Tail]),
		beforeLeft:     beforeLeft,
		hasBeforeLeft:  hasBeforeLeft,
		beforeRight:    beforeRight,
		hasBeforeRight: hasBeforeRight,
		afterLeft:      afterLeft,
		hasAfterLeft:   hasAfterLeft,
		afterRight:     afterRight,
		hasAfterRight:  hasAfterRight,
	}, nil
}

func entryNeighbors(entries []Entry, source []byte, index int) (entryLocator, bool, entryLocator, bool, error) {
	var left, right entryLocator
	hasLeft, hasRight := index > 0, index+1 < len(entries)
	if hasLeft {
		var err error
		left, err = entryLocatorFor(entries[index-1], source)
		if err != nil {
			return entryLocator{}, false, entryLocator{}, false, err
		}
	}
	if hasRight {
		var err error
		right, err = entryLocatorFor(entries[index+1], source)
		if err != nil {
			return entryLocator{}, false, entryLocator{}, false, err
		}
	}
	return left, hasLeft, right, hasRight, nil
}

func moveEntryInList(entries []Entry, from, to int) []Entry {
	result := append([]Entry(nil), entries...)
	moved := result[from]
	if from < to {
		copy(result[from:to], result[from+1:to+1])
	} else if from > to {
		copy(result[to+1:from+1], result[to:from])
	}
	result[to] = moved
	return result
}

func moveDestinationIndex(entries []Entry, source []byte, left entryLocator, hasLeft bool, right entryLocator, hasRight bool) (int, error) {
	leftIndex, rightIndex := -1, -1
	if hasLeft {
		var err error
		leftIndex, err = findEntryLocator(entries, source, left)
		if err != nil {
			return 0, err
		}
	}
	if hasRight {
		var err error
		rightIndex, err = findEntryLocator(entries, source, right)
		if err != nil {
			return 0, err
		}
	}
	switch {
	case hasLeft && hasRight:
		if rightIndex != leftIndex+1 {
			return 0, ErrPatchTargetConflict
		}
		return rightIndex, nil
	case hasLeft:
		if leftIndex != len(entries)-1 {
			return 0, ErrPatchTargetConflict
		}
		return len(entries), nil
	case hasRight:
		if rightIndex != 0 {
			return 0, ErrPatchTargetConflict
		}
		return 0, nil
	default:
		if len(entries) != 0 {
			return 0, ErrPatchTargetConflict
		}
		return 0, nil
	}
}

func (p identifiedEntryMovePatch) ApplyToCueSource(source []byte) ([]byte, error) {
	book, entries, err := entriesFromSource(source)
	if err != nil {
		return nil, err
	}
	from, found, err := findEntryIdentity(book, entries, p.target)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, ErrPatchTargetConflict
	}
	r, err := entries[from].originalSourceRange()
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(source[r.Head:r.Tail], p.before) {
		return nil, ErrPatchTargetConflict
	}
	if err := entryHasExpectedNeighbors(entries, source, from, p.afterLeft, p.hasAfterLeft, p.afterRight, p.hasAfterRight); err == nil {
		return source, nil
	} else if err != ErrPatchTargetConflict {
		return nil, err
	}
	if err := entryHasExpectedNeighbors(entries, source, from, p.beforeLeft, p.hasBeforeLeft, p.beforeRight, p.hasBeforeRight); err != nil {
		return nil, err
	}

	remaining := append([]Entry(nil), entries[:from]...)
	remaining = append(remaining, entries[from+1:]...)
	to, err := moveDestinationIndex(remaining, source, p.afterLeft, p.hasAfterLeft, p.afterRight, p.hasAfterRight)
	if err != nil {
		return nil, err
	}
	return moveEntriesByIndex(source, entries, from, to)
}

func (p identifiedEntryMovePatch) Difference() ByteAnchor {
	return ByteAnchor{Content: []byte(p.target.value)}
}

func (p identifiedEntryMovePatch) Invert() Patch {
	return identifiedEntryMovePatch{
		target:         p.target,
		before:         bytes.Clone(p.before),
		beforeLeft:     cloneEntryLocator(p.afterLeft),
		hasBeforeLeft:  p.hasAfterLeft,
		beforeRight:    cloneEntryLocator(p.afterRight),
		hasBeforeRight: p.hasAfterRight,
		afterLeft:      cloneEntryLocator(p.beforeLeft),
		hasAfterLeft:   p.hasBeforeLeft,
		afterRight:     cloneEntryLocator(p.beforeRight),
		hasAfterRight:  p.hasBeforeRight,
	}
}

func (p identifiedEntryMovePatch) patchTargetIdentity() (entryIdentity, bool) {
	return p.target, p.target.valid()
}

type identifiedEntrySwapPatch struct {
	a            entryIdentity
	b            entryIdentity
	aBefore      []byte
	bBefore      []byte
	aBeforeFirst bool
}

func newIdentifiedEntrySwapPatch(book Book, a, b entryIdentity) (Patch, error) {
	if a.fieldName != b.fieldName {
		return nil, ErrPatchTargetConflict
	}
	parsedBook, entries, err := entriesFromSource(book.source)
	if err != nil {
		return nil, err
	}
	aIndex, aFound, err := findEntryIdentity(parsedBook, entries, a)
	if err != nil {
		return nil, err
	}
	bIndex, bFound, err := findEntryIdentity(parsedBook, entries, b)
	if err != nil {
		return nil, err
	}
	if !aFound || !bFound {
		return nil, ErrPatchTargetConflict
	}
	if aIndex == bIndex {
		return nothingPatch{A: entries[aIndex].sourceAnchor, B: entries[bIndex].sourceAnchor}, nil
	}
	aRange, err := entries[aIndex].originalSourceRange()
	if err != nil {
		return nil, err
	}
	bRange, err := entries[bIndex].originalSourceRange()
	if err != nil {
		return nil, err
	}
	return identifiedEntrySwapPatch{
		a:            a,
		b:            b,
		aBefore:      bytes.Clone(book.source[aRange.Head:aRange.Tail]),
		bBefore:      bytes.Clone(book.source[bRange.Head:bRange.Tail]),
		aBeforeFirst: aIndex < bIndex,
	}, nil
}

func (p identifiedEntrySwapPatch) ApplyToCueSource(source []byte) ([]byte, error) {
	book, entries, err := entriesFromSource(source)
	if err != nil {
		return nil, err
	}
	aIndex, aFound, err := findEntryIdentity(book, entries, p.a)
	if err != nil {
		return nil, err
	}
	bIndex, bFound, err := findEntryIdentity(book, entries, p.b)
	if err != nil {
		return nil, err
	}
	if !aFound || !bFound || aIndex == bIndex {
		return nil, ErrPatchTargetConflict
	}
	aRange, err := entries[aIndex].originalSourceRange()
	if err != nil {
		return nil, err
	}
	bRange, err := entries[bIndex].originalSourceRange()
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(source[aRange.Head:aRange.Tail], p.aBefore) || !bytes.Equal(source[bRange.Head:bRange.Tail], p.bBefore) {
		return nil, ErrPatchTargetConflict
	}
	aBeforeFirst := aIndex < bIndex
	if aBeforeFirst != p.aBeforeFirst {
		return source, nil
	}
	change, err := swapByteRanges(source, aRange, bRange)
	if err != nil {
		return nil, err
	}
	return change.ApplyToCueSource(source)
}

func (p identifiedEntrySwapPatch) Difference() ByteAnchor {
	return ByteAnchor{Content: []byte(p.a.value)}
}

func (p identifiedEntrySwapPatch) Invert() Patch {
	return identifiedEntrySwapPatch{
		a:            p.a,
		b:            p.b,
		aBefore:      bytes.Clone(p.aBefore),
		bBefore:      bytes.Clone(p.bBefore),
		aBeforeFirst: !p.aBeforeFirst,
	}
}

func (p identifiedEntrySwapPatch) patchTargetIdentity() (entryIdentity, bool) {
	return p.a, p.a.valid()
}
