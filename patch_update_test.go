package cuebook

import (
	"bytes"
	"errors"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/cuecontext"
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
	for value := range EachValue(document) {
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

func TestUpdateReturnsEntryNotStructError(t *testing.T) {
	entry := cuecontext.New().CompileBytes([]byte(`"not a struct"`))
	tests := []struct {
		name string
		call func() error
	}{
		{
			name: "update field",
			call: func() error {
				_, err := UpdateFieldValue(nil, entry, cue.Value{}, "value")
				return err
			},
		},
		{
			name: "merge field values",
			call: func() error {
				_, err := MergeFieldValues(nil, entry, nil)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(); !errors.Is(err, ErrEntryNotStruct) {
				t.Fatalf("error = %v, want %v", err, ErrEntryNotStruct)
			}
		})
	}
}

func TestUpdateReturnsTargetFieldNotStructFieldError(t *testing.T) {
	ctx := cuecontext.New()
	entry := ctx.CompileBytes([]byte(`{name: "entry"}`))
	field := ctx.CompileBytes([]byte(`"value"`))

	_, err := UpdateFieldValue(nil, entry, field, "updated")
	if !errors.Is(err, ErrTargetFieldNotStructField) {
		t.Fatalf("error = %v, want %v", err, ErrTargetFieldNotStructField)
	}
}

func TestMergeFieldValuesWrapsFormattingError(t *testing.T) {
	entry := cuecontext.New().CompileString(`{Name: "" @cuebook(default=missingTestDefault)}`)
	if err := entry.Err(); err != nil {
		t.Fatal(err)
	}

	_, err := MergeFieldValues(nil, entry, map[string]string{"Name": ""})
	if err == nil {
		t.Fatal("MergeFieldValues() error = nil, want a formatting error")
	}
	if !errors.Is(err, ErrUnableToFormatFieldValue) {
		t.Fatalf("error = %v, want to include %v", err, ErrUnableToFormatFieldValue)
	}
	if got, want := err.Error(), "failed to format field value: no such default function: missingTestDefault"; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}

	var code PatchError
	if !errors.As(err, &code) || code != ErrUnableToFormatFieldValue {
		t.Fatalf("error code = %v, want %v", code, ErrUnableToFormatFieldValue)
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
	entry, err := NewEntry(target)
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
	entry, err := NewEntry(target)
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
}

func TestMergeFieldValuesAddsNewFieldsInStableOrder(t *testing.T) {
	source := []byte(`[{}]`)
	document := cuecontext.New().CompileBytes(source)
	if err := document.Err(); err != nil {
		t.Fatal(err)
	}
	entry := document.LookupPath(cue.MakePath(cue.Index(0)))
	change, err := MergeFieldValues(source, entry, map[string]string{
		"zField": "last",
		"aField": "first",
	})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := change.ApplyToCueSource(source)
	if err != nil {
		t.Fatal(err)
	}

	aIndex := bytes.Index(updated, []byte("aField"))
	zIndex := bytes.Index(updated, []byte("zField"))
	if aIndex < 0 || zIndex < 0 || aIndex >= zIndex {
		t.Fatalf("new fields are not in stable sorted order: %s", updated)
	}
}
