package cuebook

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"hash/fnv"
	"io"
	"iter"
	"log/slog"
	"math/big"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"
)

// Patch is a reversible source change that can be applied to a CUE document.
type Patch interface {
	ApplyToCueSource(original []byte) (updated []byte, err error)
	Difference() ByteAnchor
	Invert() Patch
}

// Result contains a validated document and the source revision produced by a patch.
type Result struct {
	Document   Book
	Source     []byte
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

func (r Result) BottomChangeIndex(since Result) (i int) {
	nextOlder, close := iter.Pull[cue.Value](EachValue(since.Document.Value))
	defer close()
	index := -1
	for entry, err := range r.Document.EachEntry() {
		index++
		if err != nil {
			continue
		}
		olderValue, ok := nextOlder()
		if !ok {
			return index
		}
		current, err := NewByteRange(entry.Value)
		if err != nil || !validByteRange(current, r.Source) {
			continue
		}
		older, err := NewByteRange(olderValue)
		if err != nil || !validByteRange(older, since.Source) {
			continue
		}
		if !bytes.Equal(r.Source[current.Head:current.Tail], since.Source[older.Head:older.Tail]) {
			return index
		}
	}
	return -1
}

// Match returns the entry in this result identified by the last change's
// difference anchor. For insertions and deletions, that anchor may include
// surrounding bytes; an entry contained by the matched range is returned.
func (r Result) Match(difference []byte) (entry Entry, ok bool) {
	if r.LastChange == nil || len(difference) == 0 {
		return entry, false
	}

	anchor := ByteAnchor{Content: difference}
	lastDifference := r.LastChange.Difference()
	if bytes.Equal(lastDifference.Content, difference) {
		anchor = lastDifference
	}
	target, err := anchor.Match(r.Source)
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

// Commit applies a patch to fresh source, validates the result, then atomically
// replaces targetPath. swapPath is the staging directory; it must be on the same
// filesystem as targetPath for the atomic rename to succeed. An empty swapPath
// uses the target directory.
func Commit(targetPath, swapPath string, p Patch) (r Result, err error) {
	source, err := os.ReadFile(targetPath)
	if err != nil {
		return r, err
	}
	updated, err := p.ApplyToCueSource(source)
	if err != nil {
		return r, err
	}
	r.Source = updated
	r.Document, err = New(r.Source)
	if err != nil {
		return r, err
	}

	hash := fnv.New64()
	if _, err = io.Copy(hash, bytes.NewReader(r.Source)); err != nil {
		return r, fmt.Errorf("unable to hash patch result: %w", err)
	}
	r.Hash = hash.Sum64()
	r.LastChange = p

	targetInfo, err := os.Stat(targetPath)
	if err != nil {
		return r, err
	}
	if swapPath == "" {
		swapPath = filepath.Dir(targetPath)
	}
	if err = replaceFileAtomically(targetPath, swapPath, r.Source, targetInfo.Mode().Perm(), os.Rename); err != nil {
		return r, err
	}
	return r, nil
}

func replaceFileAtomically(
	targetPath string,
	swapPath string,
	content []byte,
	mode os.FileMode,
	rename func(oldPath, newPath string) error,
) (err error) {
	tempFile, err := os.CreateTemp(swapPath, "."+filepath.Base(targetPath)+"-*.tmp")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	tempExists := true
	fileClosed := false
	defer func() {
		if !fileClosed {
			_ = tempFile.Close()
		}
		if tempExists {
			_ = os.Remove(tempPath)
		}
	}()

	written, writeErr := tempFile.Write(content)
	if writeErr == nil && written != len(content) {
		writeErr = io.ErrShortWrite
	}
	if writeErr != nil {
		return fmt.Errorf("unable to write patch result: %w", writeErr)
	}
	if err = tempFile.Chmod(mode); err != nil {
		return fmt.Errorf("unable to preserve target permissions: %w", err)
	}
	if err = tempFile.Sync(); err != nil {
		return fmt.Errorf("unable to sync patch result: %w", err)
	}
	if err = tempFile.Close(); err != nil {
		fileClosed = true
		return fmt.Errorf("unable to close patch result: %w", err)
	}
	fileClosed = true
	if err = rename(tempPath, targetPath); err != nil {
		return err
	}
	tempExists = false
	return nil
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
