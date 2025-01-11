package patch

import (
	"bytes"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/dkotik/cuebook"
)

func TestUpdateStructInList(t *testing.T) {
	source := []byte(`[
		{ one: "ok" },
		{	another: "ok" },
		{	another: "ok" },
		{	two: "ok" },
		{	two: "ok" },
		{	two: "ok" },
		{	two: "ok" }
]`)
	document := cuecontext.New().CompileBytes(source)
	err := document.Err()
	if err != nil {
		t.Fatal(err)
	}

	// // last := document.LookupPath(cue.MakePath(cue.Index(2)))
	replacement := []byte(`{ replacement: "replacement"}`)
	for value := range cuebook.EachValue(document) {
		if err = value.Err(); err != nil {
			t.Fatal(err)
		}
		patch, err := ReplaceStructListEntry(source, value, replacement)
		if err != nil {
			t.Fatal(err)
		}
		patch = Validated(patch)
		updated, err := patch.ApplyToCueSource(source)
		if err != nil {
			t.Fatal(err)
		}
		if count := bytes.Count(updated, replacement); count != 1 {
			t.Fatal("wrong replacement count:", count)
		}
		t.Run("reverse", ensureInversible(source, patch))
	}
}
