package cuebook

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"hash/fnv"
	"io"
	"log/slog"
	"math/big"
)

// Patch is a reversible source change that can be applied to a CUE document.
type Patch interface {
	ApplyToCueSource(original []byte) (updated []byte, err error)
	Difference() ByteAnchor
	Invert() Patch
}

// Result contains a validated document and the source revision produced by a patch.
type Result struct {
	Document Book
	Source   []byte
	// LastChange is nil when applying the patch did not change the source.
	LastChange Patch
	Hash       uint64
}

func (r Result) Revision() string {
	eb := big.NewInt(0).SetUint64(r.Hash)
	return base64.RawURLEncoding.EncodeToString(eb.Bytes())
}

func (r Result) IsEqual(another Result) bool {
	return r.Hash == another.Hash
}

func (r Result) BottomChangeIndex(since Result) (int, error) {
	olderValues, err := since.Document.Value.List()
	if err != nil {
		return -1, err
	}

	index := -1
	for entry, err := range r.Document.EachEntry() {
		index++
		if err != nil {
			return -1, err
		}
		if !olderValues.Next() {
			return index, nil
		}
		olderValue := olderValues.Value()
		current, err := NewByteRange(entry.Value)
		if err != nil || !validByteRange(current, r.Source) {
			continue
		}
		older, err := NewByteRange(olderValue)
		if err != nil || !validByteRange(older, since.Source) {
			continue
		}
		if !bytes.Equal(r.Source[current.Head:current.Tail], since.Source[older.Head:older.Tail]) {
			return index, nil
		}
	}
	return -1, nil
}

// Match returns the entry in this result identified by the last change's
// difference anchor. For insertions and deletions, that anchor may include
// surrounding bytes; an entry contained by the matched range is returned.
func (r Result) Match(difference []byte) (entry Entry, ok bool) {
	if r.LastChange == nil {
		return entry, false
	}

	lastDifference := r.LastChange.Difference()
	if identified, ok := r.LastChange.(interface {
		patchTargetIdentity() (entryIdentity, bool)
	}); ok {
		if identity, hasIdentity := identified.patchTargetIdentity(); hasIdentity {
			if !bytes.Equal(lastDifference.Content, difference) {
				return entry, false
			}
			for current, err := range r.Document.EachEntry() {
				if err == nil && current.identifierFieldName == identity.fieldName && current.UUID == identity.value {
					return current, true
				}
			}
			return entry, false
		}
	}
	if len(difference) == 0 {
		return entry, false
	}

	anchor := ByteAnchor{Content: difference}
	if bytes.Equal(lastDifference.Content, difference) {
		anchor = lastDifference
	}
	target, err := anchor.MatchUnique(r.Source)
	if err != nil {
		return entry, false
	}
	for current, err := range r.Document.EachEntry() {
		if err != nil {
			continue
		}
		currentRange, err := NewByteRange(current.Value)
		if err != nil || !validByteRange(currentRange, r.Source) {
			continue
		}
		if currentRange.Head >= target.Head && currentRange.Tail <= target.Tail {
			return current, true
		}
	}
	return entry, false
}

func (r Result) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("revision", r.Revision()),
		slog.Any("patch", r.LastChange),
	)
}

// Commit reads the named file from fs, applies a patch to that fresh source,
// validates the result, then writes it back. Atomicity of the write is provided
// by the [FileSystem]; the one from [NewFileSystem] renames a temporary file
// over the original and preserves its permissions.
func Commit(ctx context.Context, fs FileSystem, name string, p Patch) (r Result, err error) {
	source, err := fs.ReadFile(ctx, name)
	if err != nil {
		return r, err
	}
	updated, err := p.ApplyToCueSource(source)
	if err != nil {
		return r, err
	}
	r.Source = updated
	changed := !bytes.Equal(source, updated)
	r.Document, err = New(r.Source)
	if err != nil {
		return r, err
	}

	hash := fnv.New64()
	if _, err = io.Copy(hash, bytes.NewReader(r.Source)); err != nil {
		return r, fmt.Errorf("unable to hash patch result: %w", err)
	}
	r.Hash = hash.Sum64()

	if changed {
		r.LastChange = p
		if err = fs.WriteFile(ctx, name, r.Source); err != nil {
			return r, err
		}
	}
	return r, nil
}

type nothingPatch struct {
	A, B ByteAnchor
}

func (p nothingPatch) Difference() ByteAnchor {
	return p.B
}

func (p nothingPatch) ApplyToCueSource(source []byte) ([]byte, error) {
	return source, nil
}

func (p nothingPatch) Invert() Patch {
	return nothingPatch{A: p.B, B: p.A}
}

// Validated wraps a patch and validates each applied result as a CUE book.
func Validated(p Patch) Patch {
	return validatedPatch{Patch: p}
}

type validatedPatch struct {
	Patch
}

func (p validatedPatch) ApplyToCueSource(source []byte) ([]byte, error) {
	result, err := p.Patch.ApplyToCueSource(source)
	if err != nil {
		return result, err
	}
	if _, err := New(result); err != nil {
		return result, err
	}
	return result, nil
}

func (p validatedPatch) Invert() Patch {
	return validatedPatch{Patch: p.Patch.Invert()}
}

func (p validatedPatch) patchTargetIdentity() (entryIdentity, bool) {
	identified, ok := p.Patch.(interface {
		patchTargetIdentity() (entryIdentity, bool)
	})
	if !ok {
		return entryIdentity{}, false
	}
	return identified.patchTargetIdentity()
}
