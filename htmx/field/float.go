package field

import "cuelang.org/go/cue"

type float struct{}

func (float) MarshallText(value cue.Value) ([]byte, error) {
	return marshalNumber(value)
}
