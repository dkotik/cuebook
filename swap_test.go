package cuebook

import (
	"bytes"
	"strings"
	"testing"
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
	document, err := New(source)
	if err != nil {
		t.Fatal(err)
	}

	var entries []EntryRef
	for entry, err := range document.EachEntryRef() {
		if err != nil {
			t.Fatal(err)
		}
		entries = append(entries, entry)
	}
	for _, a := range entries {
		for _, b := range entries {
			patch, err := document.SwapEntries(a, b)
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
}

func TestMoveEntryPatchAllowsUnrelatedEditsOutsideItsTarget(t *testing.T) {
	source := []byte("// " + strings.Repeat("a", 100) + "\n" + strings.Repeat("\n", 80) + `[{Name: "one"}, {Name: "two"}]` + strings.Repeat("\n", 80) + "// trailing comment\n")
	book, err := New(source)
	if err != nil {
		t.Fatal(err)
	}
	patch, err := book.MoveEntry(0, 1)
	if err != nil {
		t.Fatal(err)
	}

	freshSource := bytes.Replace(source, []byte("a"), []byte("b"), 1)
	updated, err := patch.ApplyToCueSource(freshSource)
	if err != nil {
		t.Fatalf("move failed after an unrelated distant edit: %v", err)
	}
	if !bytes.Contains(updated, []byte("// b")) {
		t.Fatalf("unrelated edit was lost: %q", updated)
	}
	updatedBook, err := New(updated)
	if err != nil {
		t.Fatal(err)
	}
	first, err := updatedBook.GetEntryByIndex(0)
	if err != nil {
		t.Fatal(err)
	}
	if got := first.Title; got != "two" {
		t.Fatalf("first moved entry title = %q, want %q", got, "two")
	}
}

func TestSwapDifference(t *testing.T) {
	source := []byte(`[
		{ one: "ok" },
		{	two: "ok" },
		{	two: "ok" }
]`)
	document, err := New(source)
	if err != nil {
		t.Fatal(err)
	}

	a, err := document.GetEntryRef(1)
	if err != nil {
		t.Fatal(err)
	}
	b, err := document.GetEntryRef(2)
	if err != nil {
		t.Fatal(err)
	}

	p, err := document.SwapEntries(a, b)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(p.Difference().PreceedingDuplicates)
	t.Log(p.Invert().Difference().PreceedingDuplicates)
	// t.Fatal("impl")
}
