package cuebook

import (
	"bytes"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/parser"
)

// SwapEntries creates a patch that exchanges two entries in this book.
func (book Book) SwapEntries(a, b EntryRef) (Patch, error) {
	if book.Value.Kind() != cue.ListKind {
		return nil, ErrSourceIsNotList
	}
	if _, err := book.patchContext(a); err != nil {
		return nil, err
	}
	if _, err := book.patchContext(b); err != nil {
		return nil, err
	}
	aIdentity, aIdentified := a.entryIdentity()
	bIdentity, bIdentified := b.entryIdentity()
	if aIdentified && bIdentified {
		return newIdentifiedEntrySwapPatch(book, aIdentity, bIdentity)
	}
	aRange, err := a.sourceAnchor.MatchUnique(book.source)
	if err != nil {
		return nil, err
	}
	bRange, err := b.sourceAnchor.MatchUnique(book.source)
	if err != nil {
		return nil, err
	}
	return swapByteRanges(book.source, aRange, bRange)
}

// MoveEntry creates one immutable transition to the requested entry order.
// Unlike composing swaps, the resulting patch can be retried idempotently.
func (book Book) MoveEntry(from, to int) (Patch, error) {
	if book.Value.Kind() != cue.ListKind {
		return nil, ErrSourceIsNotList
	}
	length, err := book.Len()
	if err != nil {
		return nil, err
	}
	if from < 0 || from >= length || to < 0 || to >= length {
		return nil, ErrBookEntryIndexOutOfRange
	}
	originalEntry, err := book.GetEntryRef(from)
	if err != nil {
		return nil, err
	}
	if from == to {
		return nothingPatch{A: originalEntry.sourceAnchor, B: originalEntry.sourceAnchor}, nil
	}
	if _, identified := originalEntry.entryIdentity(); identified {
		parsedBook, entries, err := entriesFromSource(book.source)
		if err != nil {
			return nil, err
		}
		return newIdentifiedEntryMovePatch(parsedBook, entries, from, to)
	}

	entries := make([]ByteRange, 0, length)
	for entry, err := range book.EachEntryRef() {
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry.sourceRange)
	}

	targetRange := ByteRange{
		Head: entries[min(from, to)].Head,
		Tail: entries[max(from, to)].Tail,
	}
	current := bytes.Clone(book.source)
	direction := 1
	if from > to {
		direction = -1
	}
	for index := from; index != to; index += direction {
		next := index + direction
		change, err := swapByteRanges(current, entries[index], entries[next])
		if err != nil {
			return nil, err
		}
		current, err = change.ApplyToCueSource(current)
		if err != nil {
			return nil, err
		}
		newA, newB := rangesAfterSwap(entries[index], entries[next])
		entries[index], entries[next] = newB, newA
	}
	if bytes.Equal(current, book.source) {
		return nothingPatch{A: originalEntry.sourceAnchor, B: originalEntry.sourceAnchor}, nil
	}

	movedAnchor := entries[to].Anchor(current)
	change, err := replaceByteRangeWithDifference(
		book.source,
		targetRange,
		current[targetRange.Head:targetRange.Tail],
		movedAnchor,
	)
	if err != nil {
		return nil, err
	}
	change.BeforeDifferenceAnchor = cloneByteAnchor(originalEntry.sourceAnchor)
	return change, nil
}

func moveEntriesByIndex(source []byte, entries []EntryRef, from, to int) ([]byte, error) {
	byteRanges := make([]ByteRange, len(entries))
	for index, entry := range entries {
		byteRanges[index] = entry.sourceRange
	}
	current := bytes.Clone(source)
	direction := 1
	if from > to {
		direction = -1
	}
	for index := from; index != to; index += direction {
		next := index + direction
		change, err := swapByteRanges(current, byteRanges[index], byteRanges[next])
		if err != nil {
			return nil, err
		}
		current, err = change.ApplyToCueSource(current)
		if err != nil {
			return nil, err
		}
		newA, newB := rangesAfterSwap(byteRanges[index], byteRanges[next])
		byteRanges[index], byteRanges[next] = newB, newA
	}
	return current, nil
}

func rangesAfterSwap(a, b ByteRange) (ByteRange, ByteRange) {
	aWasEarlier := a.Head < b.Head
	earlier, later := a, b
	if !aWasEarlier {
		earlier, later = b, a
	}
	gap := later.Head - earlier.Tail
	if aWasEarlier {
		bHead := earlier.Head
		aHead := bHead + later.Tail - later.Head + gap
		return ByteRange{Head: aHead, Tail: aHead + a.Tail - a.Head},
			ByteRange{Head: bHead, Tail: bHead + b.Tail - b.Head}
	}
	aHead := earlier.Head
	bHead := aHead + later.Tail - later.Head + gap
	return ByteRange{Head: aHead, Tail: aHead + a.Tail - a.Head},
		ByteRange{Head: bHead, Tail: bHead + b.Tail - b.Head}
}

func (e entryPatchContext) swapFields(a, b string) (Patch, error) {
	if e.Value.IncompleteKind() != cue.StructKind {
		return nil, ErrEntryNotStruct
	}
	entryRange, err := e.originalSourceRange()
	if err != nil {
		return nil, err
	}
	patch, err := swapFieldsAtRange(e.source, entryRange, a, b)
	if err != nil {
		return nil, err
	}
	if e.identifierFieldName != "" && a != b && (a == e.identifierFieldName || b == e.identifierFieldName) {
		return nil, ErrEntryIdentifierImmutable
	}
	return identifyEntryPatch(e, patch)
}

// swapFieldsAtRange exchanges two written field declarations, including their
// labels and attributes. Separators and comments stay in place.
func swapFieldsAtRange(source []byte, entryRange ByteRange, a, b string) (Patch, error) {
	if !validByteRange(entryRange, source) {
		return nil, ErrByteRangeNotFound
	}
	expression, err := parser.ParseExpr("", source[entryRange.Head:entryRange.Tail])
	if err != nil {
		return nil, err
	}
	fields, ok := expression.(*ast.StructLit)
	if !ok {
		return nil, ErrEntryNotStruct
	}
	aRange, err := fieldByteRange(fields, a, entryRange.Head)
	if err != nil {
		return nil, err
	}
	bRange, err := fieldByteRange(fields, b, entryRange.Head)
	if err != nil {
		return nil, err
	}
	return swapByteRanges(source, aRange, bRange)
}

func fieldByteRange(fields *ast.StructLit, label string, offset int) (ByteRange, error) {
	for _, element := range fields.Elts {
		field, ok := element.(*ast.Field)
		if !ok {
			continue
		}
		name, _, err := ast.LabelName(field.Label)
		if err != nil || name != label {
			continue
		}
		return ByteRange{
			Head: offset + field.Pos().Offset(),
			Tail: offset + field.End().Offset(),
		}, nil
	}
	return ByteRange{}, ErrTargetFieldNotStructField
}

func swapByteRanges(source []byte, aRange, bRange ByteRange) (Patch, error) {
	if !validByteRange(aRange, source) || !validByteRange(bRange, source) {
		return nil, ErrByteRangeNotFound
	}
	if bytes.Equal(source[aRange.Head:aRange.Tail], source[bRange.Head:bRange.Tail]) {
		return nothingPatch{A: aRange.Anchor(source), B: bRange.Anchor(source)}, nil
	}

	aWasEarlier := aRange.Head < bRange.Head
	earlier, later := aRange, bRange
	if !aWasEarlier {
		earlier, later = bRange, aRange
	}
	if earlier.Tail > later.Head {
		return nil, ErrByteRangesOverlap
	}

	afterSegment := make([]byte, 0, later.Tail-earlier.Head)
	afterSegment = append(afterSegment, source[later.Head:later.Tail]...)
	afterSegment = append(afterSegment, source[earlier.Tail:later.Head]...)
	afterSegment = append(afterSegment, source[earlier.Head:earlier.Tail]...)
	afterSource, err := spliceRange(source, ByteRange{Head: earlier.Head, Tail: later.Tail}, afterSegment)
	if err != nil {
		return nil, err
	}

	targetAHead := earlier.Head
	if aWasEarlier {
		targetAHead += later.Tail - later.Head + later.Head - earlier.Tail
	}
	targetARange := ByteRange{Head: targetAHead, Tail: targetAHead + aRange.Tail - aRange.Head}
	difference := targetARange.Anchor(afterSource)
	return replaceByteRangeWithDifference(
		source,
		ByteRange{Head: earlier.Head, Tail: later.Tail},
		afterSegment,
		difference,
	)
}
