package cuebook

import (
	"crypto/sha256"
	"errors"
	"iter"
	"slices"
)

// EntryRef identifies an entry in the source revision of a Book. It keeps
// source-location data separate from Entry's semantic representation.
// References are created by Book.GetEntryRef or Book.EachEntryRef and must be
// used with a Book containing the same source revision.
type EntryRef struct {
	entry                   Entry
	sourceFingerprint       [32]byte
	sourceRange             ByteRange
	hasSourceRange          bool
	precedingSourceRange    ByteRange
	hasPrecedingSourceRange bool
	sourceAnchor            ByteAnchor
}

// Entry returns the semantic entry described by this reference.
func (ref EntryRef) Entry() Entry {
	entry := ref.entry
	entry.Fields = slices.Clone(entry.Fields)
	entry.Details = slices.Clone(entry.Details)
	entry.FieldDefinitions = slices.Clone(entry.FieldDefinitions)
	return entry
}

func (ref EntryRef) entryIdentity() (entryIdentity, bool) {
	return ref.entry.entryIdentity()
}

func (ref EntryRef) originalSourceRange() (ByteRange, error) {
	if !ref.hasSourceRange || ref.sourceRange.Head < 0 || ref.sourceRange.Head > ref.sourceRange.Tail {
		return ByteRange{}, ErrByteRangeNotFound
	}
	return ref.sourceRange, nil
}

// EachEntryRef iterates over entries together with source locators bound to
// this Book's source revision. Use EachEntry for semantic-only iteration.
func (book Book) EachEntryRef() iter.Seq2[EntryRef, error] {
	return func(yield func(EntryRef, error) bool) {
		fingerprint := sha256.Sum256(book.source)
		var precedingRange ByteRange
		hasPrecedingRange := false
		for entry, entryErr := range book.EachEntry() {
			ref := EntryRef{
				entry:                   entry,
				sourceFingerprint:       fingerprint,
				precedingSourceRange:    precedingRange,
				hasPrecedingSourceRange: hasPrecedingRange,
			}
			rangeInSource, rangeErr := NewByteRange(entry.Value)
			if rangeErr == nil {
				var anchor ByteAnchor
				anchor, rangeErr = anchorForRange(rangeInSource, book.source)
				if rangeErr == nil {
					ref.sourceRange = rangeInSource
					ref.hasSourceRange = true
					ref.sourceAnchor = anchor
				}
			}
			if entryErr != nil || rangeErr != nil {
				if entryErr != nil && rangeErr != nil {
					entryErr = errors.Join(entryErr, rangeErr)
				} else if entryErr == nil {
					entryErr = rangeErr
				}
				if !yield(ref, entryErr) {
					return
				}
				return
			}
			if !yield(ref, nil) {
				return
			}
			precedingRange = rangeInSource
			hasPrecedingRange = true
		}
	}
}

// GetEntryRef returns an entry together with its source locator.
func (book Book) GetEntryRef(atIndex int) (EntryRef, error) {
	if atIndex < 0 {
		return EntryRef{}, ErrBookEntryIndexOutOfRange
	}
	index := 0
	for ref, err := range book.EachEntryRef() {
		if err != nil {
			return ref, err
		}
		if index == atIndex {
			return ref, nil
		}
		index++
	}
	return EntryRef{}, ErrBookEntryIndexOutOfRange
}

type entryPatchContext struct {
	Entry
	source                  []byte
	sourceRange             ByteRange
	hasSourceRange          bool
	precedingSourceRange    ByteRange
	hasPrecedingSourceRange bool
	sourceAnchor            ByteAnchor
}

func (entry entryPatchContext) originalSourceRange() (ByteRange, error) {
	if !entry.hasSourceRange || !validByteRange(entry.sourceRange, entry.source) {
		return ByteRange{}, ErrByteRangeNotFound
	}
	return entry.sourceRange, nil
}

func (book Book) patchContext(ref EntryRef) (entryPatchContext, error) {
	if ref.sourceFingerprint != sha256.Sum256(book.source) {
		return entryPatchContext{}, ErrPatchTargetConflict
	}
	rangeInSource, err := ref.originalSourceRange()
	if err != nil || !validByteRange(rangeInSource, book.source) {
		return entryPatchContext{}, ErrByteRangeNotFound
	}
	return entryPatchContext{
		Entry:                   ref.entry,
		source:                  book.source,
		sourceRange:             rangeInSource,
		hasSourceRange:          true,
		precedingSourceRange:    ref.precedingSourceRange,
		hasPrecedingSourceRange: ref.hasPrecedingSourceRange,
		sourceAnchor:            cloneByteAnchor(ref.sourceAnchor),
	}, nil
}

// AddField creates a patch that appends a field to the referenced entry.
func (book Book) AddField(ref EntryRef, name string, kind FieldKind, value string) (Patch, error) {
	entry, err := book.patchContext(ref)
	if err != nil {
		return nil, err
	}
	return entry.addField(name, kind, value)
}

// UpdateFieldValue creates a patch that changes a field in the referenced entry.
func (book Book) UpdateFieldValue(ref EntryRef, fieldName, value string) (Patch, error) {
	entry, err := book.patchContext(ref)
	if err != nil {
		return nil, err
	}
	return entry.updateFieldValue(fieldName, value)
}

// RemoveFieldValue creates a patch that omits a field from the referenced entry.
func (book Book) RemoveFieldValue(ref EntryRef, fieldName string) (Patch, error) {
	entry, err := book.patchContext(ref)
	if err != nil {
		return nil, err
	}
	return entry.removeFieldValue(fieldName)
}

// MergeFieldValues creates a patch that merges values into the referenced entry.
func (book Book) MergeFieldValues(ref EntryRef, values map[string]string) (Patch, error) {
	entry, err := book.patchContext(ref)
	if err != nil {
		return nil, err
	}
	return entry.mergeFieldValues(values)
}

// MergeFieldValuesAndRemove creates a patch that merges values and removes
// fields in the referenced entry. Removal takes precedence for duplicate names.
func (book Book) MergeFieldValuesAndRemove(ref EntryRef, values map[string]string, removals []string) (Patch, error) {
	entry, err := book.patchContext(ref)
	if err != nil {
		return nil, err
	}
	return entry.mergeFieldValuesAndRemove(values, removals)
}

// ReplaceStructListEntry creates a patch that replaces the referenced entry's source.
func (book Book) ReplaceStructListEntry(ref EntryRef, replacement []byte) (Patch, error) {
	entry, err := book.patchContext(ref)
	if err != nil {
		return nil, err
	}
	return entry.replaceStructListEntry(replacement)
}

// DeleteFromStructList creates a patch that removes the referenced entry.
func (book Book) DeleteFromStructList(ref EntryRef) (Patch, error) {
	entry, err := book.patchContext(ref)
	if err != nil {
		return nil, err
	}
	return entry.deleteFromStructList()
}

// SwapFields creates a patch exchanging two written fields in the referenced entry.
func (book Book) SwapFields(ref EntryRef, a, b string) (Patch, error) {
	entry, err := book.patchContext(ref)
	if err != nil {
		return nil, err
	}
	return entry.swapFields(a, b)
}
