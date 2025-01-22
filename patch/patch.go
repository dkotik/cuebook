/*
Package patch provides reversible byte range operations for a
Cuelang source. The patch operations are capable of changing the
source, in most cases, even if it was modified by
another process after the patch had already been created.
This is usually achieved by counting preceeding duplicate byte
ranges before carrying out the intended operation.
*/
package patch

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"hash/fnv"
	"io"
	"iter"
	"math/big"
	"os"
	"path/filepath"
	"strings"

	"cuelang.org/go/cue"
	"github.com/dkotik/cuebook"
)

type Patch interface {
	ApplyToCueSource(original []byte) (updated []byte, err error)
	Difference() ByteAnchor
	Invert() Patch
}

type Result struct {
	Document   cuebook.Document
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
	nextOlder, close := iter.Pull[cue.Value](since.Document.EachValue())
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
		if err != nil {
			continue
		}
		older, err := NewByteRange(olderValue)
		if err != nil {
			continue
		}
		rawCurrent := r.Source[current.Head:current.Tail]
		rawOlder := since.Source[older.Head:older.Tail]
		if !bytes.Equal(rawCurrent, rawOlder) {
			return index
		}
	}
	return -1
}

// Commit applies a [Patch] to fresh contents of the source file after validating the changes.
// A temporary file is created first using deterministic file name. Then, the temporary file
// is renamed to overwrite the original. The additional steps ensure complete atomic operations
// that anticipate that the file may be possibly changed after the patch had already been created.
func Commit(
	targetPath string,
	swapPath string,
	p Patch,
) (r Result, err error) {
	source, err := os.ReadFile(targetPath)
	if err != nil {
		return
	}
	r.Source, err = p.ApplyToCueSource(source)
	if err != nil {
		return
	}
	r.Document, err = cuebook.New(r.Source)
	if err != nil {
		return
	}

	hash := fnv.New64()
	if _, err = io.Copy(hash, bytes.NewReader(r.Source)); err != nil {
		return r, fmt.Errorf("unable to hash patch result: %w", err)
	}
	r.Hash = hash.Sum64()

	ext := filepath.Ext(targetPath)
	temp := filepath.Join(
		swapPath,
		fmt.Sprintf(
			"%s.%s.cue",
			strings.TrimSuffix(filepath.Base(targetPath), ext),
			r.Revision(),
		),
	)
	// panic(temp)
	if err = os.WriteFile(temp, r.Source, 0700); err != nil {
		return
	}
	r.LastChange = p
	return r, os.Rename(temp, targetPath)
}
