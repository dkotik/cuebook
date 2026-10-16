package field

import (
	"cuelang.org/go/cue"
	"github.com/dkotik/cuebook/metadata"
)

type password struct{}

func (password) MarshallText(value cue.Value) ([]byte, error) {
	text, err := value.String()
	if err != nil {
		return nil, err
	}
	return []byte(quoteString(text)), nil
}

func isPassword(value cue.Value) bool {
	_, ok := metadata.GetFieldAttributes(value, "cuebook").GetFirstOf("argon2id")
	return ok
}
