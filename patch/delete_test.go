package patch

import (
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/dkotik/cuebook"
)

// TODO: multiple delete and insert cycles leave whitespace artifacts

func TestDeleteStructFromList(t *testing.T) {
	source := []byte(`[
		{ one: "ok" },
		{	another: "ok" },
		{	another: "ok" },
		{	two: "ok" },
		{	two: "ok" },
		{	two: "ok" }
]`)
	document := cuecontext.New().CompileBytes(source)
	err := document.Err()
	if err != nil {
		t.Fatal(err)
	}

	// last := document.LookupPath(cue.MakePath(cue.Index(2)))
	for value := range cuebook.EachValue(document) {
		if err = value.Err(); err != nil {
			t.Fatal(err)
		}
		patch, err := DeleteFromStructList(source, value)
		if err != nil {
			t.Fatal(err)
		}
		patch = Validated(patch)
		t.Run("reverse", ensureInversible(source, patch))
	}
}
