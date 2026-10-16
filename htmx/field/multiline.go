package field

import "cuelang.org/go/cue"

type multiline struct{}

func (multiline) MarshallText(value cue.Value) ([]byte, error) {
	text, err := value.String()
	if err != nil {
		return nil, err
	}
	return []byte(quoteMultiline(text)), nil
}

func (multiline) quote(value string) string {
	return quoteMultiline(value)
}
