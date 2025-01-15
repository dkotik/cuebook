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
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"strings"

	"cuelang.org/go/cue/cuecontext"
)

type Patch interface {
	ApplyToCueSource([]byte) ([]byte, error)
	Invert() Patch
}

// Commit applies a [Patch] to fresh contents of the source file after validating the changes.
// A temporary file is created first using deterministic file name. Then, the temporary file
// is renamed to overwrite the original. The additional steps ensure complete atomic operations
// that anticipate that the file may be possibly changed after the patch had already been created.
func Commit(
	targetPath string,
	swapPath string,
	p Patch,
) (err error) {
	source, err := os.ReadFile(targetPath)
	if err != nil {
		return err
	}
	source, err = p.ApplyToCueSource(source)
	if err != nil {
		return err
	}
	value := cuecontext.New().CompileBytes(source) // TODO: add options cuecontext...
	if err = value.Err(); err != nil {
		return err
	}
	if err = value.Validate(); err != nil {
		return err
	}
	temp := filepath.Join(
		swapPath,
		temporaryName(
			filepath.Base(targetPath),
			source,
		),
	)
	if err = os.WriteFile(temp, source, 0700); err != nil {
		return err
	}
	return os.Rename(temp, targetPath)
}

func temporaryName(name string, source []byte) string {
	ext := filepath.Ext(name)
	return fmt.Sprintf("%s.%x.cue", strings.TrimSuffix(name, ext), fnv.New32().Sum(source)[:8])
}
