package patch

import (
	"bytes"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
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

func TestUpdateStructField(t *testing.T) {
	source := []byte(`
#contact: {
	Name?: string
	Email?: string
	Notes?: string
	... // allow any additional fields
}

[...#contact]&[
		{ one: "ok" },
		{	another: "ok" },
		{
			another: "ok"
			seriously: "so far"
			difficult: true
		},
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

	target := document.LookupPath(cue.MakePath(cue.Index(2)))
	if target.Err() != nil {
		t.Fatal(target.Err())
	}
	entry, err := cuebook.NewEntry(target)
	if err != nil {
		t.Fatal(err)
	}
	field, err := entry.GetField(1)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("field:", field.Name)

	patch, err := UpdateFieldValue(source, target, field.Value, "value string sdkfjlsf lskdj;lsajfskajfsadjsakdljfadsfsakj ksadjl;sajflsaf \n\n\nkjshfadsjh")
	if err != nil {
		t.Fatal(err)
	}
	patch = Validated(patch)
	updated, err := patch.ApplyToCueSource(source)
	t.Log(string(updated))
	if err != nil {
		t.Fatal(err)
	}
	t.Run("reverse", ensureInversible(source, patch))
	// t.Fatal("check")
}

func TestUpdateStructFieldsWithMapOfValues(t *testing.T) {
	source := []byte(`
#contact: {
	ID?: string @cuebook(default=SFID?prefix=prefix_)
	Name?: string
	Email?: string
	Notes?: string
	... // allow any additional fields
}

[...#contact]&[
		{ one: "ok" },
		{	another: "ok" },
		{
			another: "ok"
			seriously: "so far"
			difficult: true
			"12": "sure"
		},
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

	target := document.LookupPath(cue.MakePath(cue.Index(2)))
	if target.Err() != nil {
		t.Fatal(target.Err())
	}
	entry, err := cuebook.NewEntry(target)
	if err != nil {
		t.Fatal(err)
	}
	field, err := entry.GetField(1)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("field:", field.Name)

	patch, err := MergeFieldValues(source, target, map[string]string{
		"ID":          "",
		"another":     "123",
		"seriously":   "545445",
		"Notes":       "important to handle \nabstract optional definitions",
		"entirelyNew": "new",
		"12":          "updated",
	})
	if err != nil {
		t.Fatal(err)
	}
	patch = Validated(patch)
	updated, err := patch.ApplyToCueSource(source)
	t.Log(string(updated))
	if err != nil {
		t.Fatal(err)
	}
	t.Run("reverse", ensureInversible(source, patch))

	document = cuecontext.New().CompileBytes(updated)
	another := document.LookupPath(cue.MakePath(cue.Index(2), cue.Label(ast.NewString("another"))))
	if another.Kind() == cue.BottomKind {
		t.Fatal("another element was not found")
	}
	value, err := another.String()
	if err != nil {
		t.Fatal("another value is not a string")
	}
	if value != "123" {
		t.Fatal("another value was not updated:", value, "vs 123")
	}

	entirelyNew := document.LookupPath(cue.MakePath(cue.Index(2), cue.Label(ast.NewString("entirelyNew"))))
	if another.Kind() == cue.BottomKind {
		t.Fatal("another element was not found")
	}
	value, err = entirelyNew.String()
	if err != nil {
		t.Fatal("another value is not a string")
	}
	if value != "new" {
		t.Fatal("another value was not updated:", value, "vs new")
	}

	// t.Fatal("check")
}
