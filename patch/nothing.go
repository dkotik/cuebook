package patch

import (
	"github.com/dkotik/cuebook"
)

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
	return nothingPatch{
		A: p.B,
		B: p.A,
	}
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
