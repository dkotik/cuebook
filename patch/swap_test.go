package patch

import (
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/dkotik/cuebook"
)

func TestSwapEntries(t *testing.T) {
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

	for a := range cuebook.EachValue(document) {
		if a.Err() != nil {
			t.Fatal("unable to parse entry:", err)
		}
		for b := range cuebook.EachValue(document) {
			patch, err := SwapEntries(source, a, b)
			if err != nil {
				t.Fatal(err)
			}
			// patch = Validated(patch)
			result, err := patch.ApplyToCueSource(source)
			if err != nil {
				t.Fatal(err)
			}
			t.Log(string(result))
			// replacement := []byte(`{ replacement: "replacement"}`)
			// for value := range cuebook.EachValue(document) {
			t.Run("reverse", ensureInversible(source, patch))
		}
	}
	// a := document.LookupPath(cue.MakePath(cue.Index(0)))
	// b := document.LookupPath(cue.MakePath(cue.Index(5)))

}
