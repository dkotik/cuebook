package cuebook

import (
	"io"
	"os"

	"cuelang.org/go/cue/cuecontext"
)

// UUID is the name of the map field in which UUID is stored on the disk.
const UUID = "ID"

func parse(b []byte, target interface{}) (err error) {
	ctx := cuecontext.New()
	tree := ctx.CompileBytes(b)
	if err = tree.Err(); err != nil {
		return
	}
	return tree.Value().Decode(&target)
}

func loadEntry(source, uuid string) (v map[string]interface{}, err error) {
	b, err := os.ReadFile(source)
	if err != nil {
		return
	}
	list := make([]map[string]interface{}, 0)
	if err = parse(b, list); err != nil {
		return
	}

	for _, m := range list {
		val, ok := m[UUID].(string)
		if !ok {
			continue
		}
		if val == uuid {
			return m, nil
		}
	}
	return nil, io.EOF
}
