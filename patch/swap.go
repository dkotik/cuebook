package patch

type swapPatch struct{}

func (p swapPatch) ApplyToCueSource(source []byte) (result []byte, err error) {

	return nil, nil
}

func (p swapPatch) Invert() Patch {
	return nil
}
