package cuebook

import (
	"fmt"
	"io"
	"io/fs"
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

func write(source string, v interface{}) (err error) {
	w, err := os.OpenFile(source, os.O_CREATE|os.O_WRONLY, fs.ModePerm)
	if err != nil {
		return
	}
	defer w.Close()

	ctx := cuecontext.New()
	tree := ctx.Encode(v) // https://pkg.go.dev/cuelang.org/go/cue#Context.Encode
	if err = tree.Err(); err != nil {
		return
	}
	_, err = fmt.Fprintf(w, "%v", v)
	return
}

func load(source string) (v []map[string]interface{}, err error) {
	b, err := os.ReadFile(source)
	if err != nil {
		return
	}
	v = make([]map[string]interface{}, 0)
	err = parse(b, v)
	return
}

func loadEntry(source, uuid string) (v map[string]interface{}, err error) {
	list, err := load(source)
	if err != nil {
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
