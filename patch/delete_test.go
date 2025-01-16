package patch

import (
	"slices"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/dkotik/cuebook"
)

func TestDeleteStructFromList(t *testing.T) {
	source := []byte(`[
		{ one: "ok" },
		{	two: "ok" },
]`)
	document := cuecontext.New().CompileBytes(source)
	err := document.Err()
	if err != nil {
		t.Fatal(err)
	}

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

	// TODO: multiple delete and insert cycles leave whitespace artifacts
	last := document.LookupPath(cue.MakePath(cue.Index(1)))
	changed := slices.Clone(source)
	for range 5 {
		inverse, err := AppendToStructList(changed, last)
		if err != nil {
			t.Fatal(err)
		}
		changed, err = inverse.ApplyToCueSource(changed)
		if err != nil {
			t.Fatal(err)
		}

		patch, err := DeleteFromStructList(changed, last)
		if err != nil {
			t.Fatal(err)
		}
		changed, err = patch.ApplyToCueSource(changed)
		if err != nil {
			t.Fatal(err)
		}
	}

	// TODO: check if there is any extra whitespace
	// t.Fatal(string(changed))
}
