package field

import (
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/cuecontext"
)

func TestFieldMarshalsByCUEType(t *testing.T) {
	t.Parallel()

	compiled := cuecontext.New().CompileString(`{
		Enabled: false
		Count: 12
		Ratio: 1.5
		Plain: "line one\nline two"
		Notes: "line one\nline two" @cuebook(multiline)
	}`)
	if err := compiled.Err(); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "boolean false", path: "Enabled", want: "false"},
		{name: "integer", path: "Count", want: "12"},
		{name: "float", path: "Ratio", want: "1.5"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value := compiled.LookupPath(cue.ParsePath(test.path))
			got, err := NewField(test.name, value).MarshallText()
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != test.want {
				t.Errorf("MarshallText() = %q, want %q", got, test.want)
			}
		})
	}

	for _, test := range []struct {
		name string
		path string
	}{
		{name: "single-line string escapes newline", path: "Plain"},
		{name: "multiline string retains newline", path: "Notes"},
	} {
		t.Run(test.name, func(t *testing.T) {
			field := NewField(test.path, compiled.LookupPath(cue.ParsePath(test.path)))
			got, err := field.MarshallText()
			if err != nil {
				t.Fatal(err)
			}
			if test.path == "Plain" && strings.Contains(string(got), "\n") {
				t.Errorf("single-line string unexpectedly contains a literal newline: %q", got)
			}
			if test.path == "Notes" && !strings.Contains(string(got), "\n") {
				t.Errorf("multiline string does not contain a literal newline: %q", got)
			}

			updated, err := field.WithValue("updated one\nupdated two")
			if err != nil {
				t.Fatal(err)
			}
			literal, ok := updated.Value.(*ast.BasicLit)
			if !ok {
				t.Fatalf("updated field value has type %T, want *ast.BasicLit", updated.Value)
			}
			if test.path == "Plain" && strings.Contains(literal.Value, "\n") {
				t.Errorf("single-line field literal unexpectedly contains a literal newline: %q", literal.Value)
			}
			if test.path == "Notes" && !strings.Contains(literal.Value, "\n") {
				t.Errorf("multiline field literal does not contain a literal newline: %q", literal.Value)
			}
		})
	}
}
