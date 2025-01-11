package patch

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
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

	a := document.LookupPath(cue.MakePath(cue.Index(0)))
	b := document.LookupPath(cue.MakePath(cue.Index(5)))

	patch, err := SwapEntries(source, a, b)
	if err != nil {
		t.Fatal(err)
	}
	patch = Validated(patch)
	result, err := patch.ApplyToCueSource(source)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(result))
	// replacement := []byte(`{ replacement: "replacement"}`)
	// for value := range cuebook.EachValue(document) {
	t.Run("reverse", ensureInversible(source, patch))
}
