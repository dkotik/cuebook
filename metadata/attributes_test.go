package metadata_test

import (
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/cuecontext"
	"github.com/dkotik/cuebook/metadata"
)

func TestFormatByAttribute(t *testing.T) {
	book := cuecontext.New().CompileBytes([]byte(`{
		"key": "extra information" @cuebook(trim,argon2id,default=SFID?prefix=PREFIX)
	}`))
	err := book.Err()
	if err != nil {
		t.Fatal(err)
	}
	value := book.LookupPath(cue.MakePath(cue.Label(ast.NewString("key"))))
	if err = value.Err(); err != nil {
		t.Fatal("unable to retrieve key:", err)
	}

	result, err := metadata.FormatAccordingToAttributes(value, "input string")
	if err != nil {
		t.Fatal("unable to apply formatting:", err)
	}
	if !strings.HasPrefix(result, "$argon2id$") {
		t.Fatalf("output is not an Argon2id hash: %q", result)
	}
}
