package cuebook

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

	aValues, err := document.List()
	if err != nil {
		t.Fatal(err)
	}
	for aValues.Next() {
		a := aValues.Value()
		if err := a.Err(); err != nil {
			t.Fatal("unable to parse entry:", err)
		}
		bValues, err := document.List()
		if err != nil {
			t.Fatal(err)
		}
		for bValues.Next() {
			b := bValues.Value()
			if err := b.Err(); err != nil {
				t.Fatal("unable to parse entry:", err)
			}
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
			t.Run("reverse", ensureInversible(source, patch))
		}
	}
	// a := document.LookupPath(cue.MakePath(cue.Index(0)))
	// b := document.LookupPath(cue.MakePath(cue.Index(5)))
}

func TestSwapDifference(t *testing.T) {
	source := []byte(`[
		{ one: "ok" },
		{	two: "ok" },
		{	two: "ok" }
]`)
	document := cuecontext.New().CompileBytes(source)
	err := document.Err()
	if err != nil {
		t.Fatal(err)
	}

	a := document.LookupPath(cue.MakePath(cue.Index(1)))
	b := document.LookupPath(cue.MakePath(cue.Index(2)))

	p, err := SwapEntries(source, a, b)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(p.Difference().PreceedingDuplicates)
	t.Log(p.Invert().Difference().PreceedingDuplicates)
	// t.Fatal("impl")
}
