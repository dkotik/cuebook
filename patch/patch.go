/*
Package patch provides reversible byte range operations for a
Cuelang source. The patch operations are capable of changing the
source, in most cases, even if it was modified by
another process after the patch had already been created.
This is usually achieved by counting preceeding duplicate byte
ranges before carrying out the intended operation.
*/
package patch

import "github.com/dkotik/cuebook"

type Patch interface {
	ApplyToCueSource([]byte) ([]byte, error)
	Invert() Patch
}

func Validated(p Patch) Patch {
	return validatedPatch{Patch: p}
}

type validatedPatch struct {
	Patch
}

func (p validatedPatch) ApplyToCueSource(source []byte) (result []byte, err error) {
	result, err = p.Patch.ApplyToCueSource(source)
	if err != nil {
		return result, err
	}
	if _, err = cuebook.New(result); err != nil {
		return result, err
	}
	return result, err
}

func (p validatedPatch) Invert() Patch {
	return validatedPatch{p.Patch.Invert()}
}
