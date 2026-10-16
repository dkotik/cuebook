package cuebook

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
