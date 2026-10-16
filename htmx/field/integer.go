package field

import "cuelang.org/go/cue"

type integer struct{}

func (integer) MarshallText(value cue.Value) ([]byte, error) {
	return marshalNumber(value)
}
