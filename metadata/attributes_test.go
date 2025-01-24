package metadata_test

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/cuecontext"
	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/metadata"
)

func TestIsDetailAttribute(t *testing.T) {
	book := cuecontext.New().CompileBytes([]byte(`{
		"key": "extra information" @cuebook(detail)
	}`))
	err := book.Err()
	if err != nil {
		t.Fatal(err)
	}
	value := book.LookupPath(cue.MakePath(cue.Label(ast.NewString("key"))))
	if err = value.Err(); err != nil {
		t.Fatal("unable to retrieve key:", err)
	}
	if !metadata.IsDetailField(value) {
		t.Fatal("key is not marked as detail, but should be")
	}
}

func TestDefaultAttribute(t *testing.T) {
	t.Skip("working on another")
	book, err := cuebook.New([]byte(`[...{
		key: string | *"default"
		key2: string | *"1991-03-04" @cuebook(default="date?format=01*02")
	}]&[
		{ "key": "value" },
		{ "key2": "value2" }
	]`))
	if err != nil {
		t.Fatal(err)
	}

	last := book.Value.LookupPath(cue.MakePath(cue.Index(1)))
	if err := last.Err(); err != nil {
		t.Fatal("last entry could not be gathered:", err)
	}
	value, err := last.LookupPath(cue.MakePath(cue.Label(ast.NewString("key")))).String()
	if err != nil {
		t.Fatal("last entry.key could not be converted to string:", err)
	}
	if value != "default" {
		t.Fatalf("expected last entry to be 'default', got %s", value)
	}

	another := last.LookupPath(cue.MakePath(cue.Label(ast.NewString("key2"))))
	if err = another.Err(); err != nil {
		t.Fatal("cannot gather another field:", err)
	}
	defaultValue, ok := metadata.GetDefaultValue(another)
	if !ok {
		t.Fatal("default attribute could not be gathered")
	}
	if defaultValue != "03*04" {
		t.Fatalf("expected default attribute to be '03*04', got %s", defaultValue)
	}
}
