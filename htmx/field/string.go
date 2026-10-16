package field

import (
	"cuelang.org/go/cue"
)

type stringValue struct{}

func (stringValue) MarshallText(value cue.Value) ([]byte, error) {
	text, err := value.String()
	if err != nil {
		return nil, err
	}
	return []byte(quoteString(text)), nil
}

func (stringValue) quote(value string) string {
	return quoteString(value)
}
