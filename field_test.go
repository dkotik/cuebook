package cuebook

import (
	"bytes"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/literal"
)

const basicStruct = `{
	first: "first"
	// comment
	second: "second"
	// another comment
	third: "third"
}`

func TestFieldSubstitution(t *testing.T) {
	tree := cuecontext.New().CompileBytes([]byte(basicStruct))
	err := tree.Err()
	if err != nil {
		t.Fatal(err)
	}
	value := tree.LookupPath(cue.ParsePath("second"))
	if err = value.Err(); err != nil {
		t.Fatal(err)
	}
	b := &bytes.Buffer{}
	label, ok := value.Label()
	if !ok {
		t.Fatal("value has no label")
	}

	err = Field{
		Name:  label,
		Value: value,
	}.substituteWithValue(
		b, []byte(basicStruct), []byte(
			literal.String.WithOptionalTabIndent(0).Quote("???\n???"),
		))
	if err != nil {
		t.Fatal(err)
	}
	// t.Fatal(b.String())
}
