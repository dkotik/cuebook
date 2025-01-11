package patch

type updatePatch struct{}

func (p updatePatch) ApplyToCueSource(source []byte) (result []byte, err error) {

	return nil, nil
}

func (p updatePatch) Invert() Patch {
	return nil
}
