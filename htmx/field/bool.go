package field

import "cuelang.org/go/cue"

type boolean struct{}

func (boolean) MarshallText(value cue.Value) ([]byte, error) {
	result, err := value.Bool()
	if err != nil {
		return nil, err
	}
	if result {
		return []byte("true"), nil
	}
	return []byte("false"), nil
}
