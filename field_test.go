package cuebook

import (
	"encoding/json"
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/cuecontext"
)

func TestNewEntryPopulatesFieldDescriptionsFromPrecedingComments(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		description string
	}{
		{
			name: "single comment",
			source: `[
				{
					// The contact's display name.
					Name: "Ada",
				}
			]`,
			description: "The contact's display name.",
		},
		{
			name: "multiple comment lines",
			source: `[
				{
					// Primary contact email.
					// Use an address that can receive messages.
					Email: "ada@example.test",
				}
			]`,
			description: "Primary contact email.\nUse an address that can receive messages.",
		},
		{
			name:   "no comment",
			source: `[{Name: "Ada"}]`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			book, err := New([]byte(test.source))
			if err != nil {
				t.Fatal(err)
			}
			for entry, err := range book.EachEntry() {
				if err != nil {
					t.Fatal(err)
				}
				field, ok := entry.GetFieldByName("Name")
				if !ok {
					field, ok = entry.GetFieldByName("Email")
				}
				if !ok {
					t.Fatal("field not found")
				}
				if field.Description != test.description {
					t.Errorf("field description = %q, want %q", field.Description, test.description)
				}
				return
			}
			t.Fatal("entry not found")
		})
	}
}

func TestFieldDefault(t *testing.T) {
	t.Parallel()

	compiled := cuecontext.New().CompileString(`[...{
		Initial: string | *"starting value"
		Missing: string
	}] & [{Initial: "actual value", Missing: ""}]`)
	if err := compiled.Err(); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		path      string
		wantValue string
		wantFound bool
	}{
		{name: "CUE default", path: "Initial", wantValue: "starting value", wantFound: true},
		{name: "no default", path: "Missing"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			value := compiled.LookupPath(cue.MakePath(cue.Index(0), cue.Str(test.path)))
			field := NewField(test.path, value)
			got, ok := field.Default()
			if got != test.wantValue || ok != test.wantFound {
				t.Errorf("Default() = (%q, %t), want (%q, %t)", got, ok, test.wantValue, test.wantFound)
			}
		})
	}

	t.Run("attribute-generated default", func(t *testing.T) {
		t.Parallel()
		compiled := cuecontext.New().CompileString(`Generated: string @cuebook(default=UUID)`)
		if err := compiled.Err(); err != nil {
			t.Fatal(err)
		}
		field := NewField("Generated", compiled.LookupPath(cue.ParsePath("Generated")))
		if value, ok := field.Default(); !ok || value == "" {
			t.Errorf("Default() = (%q, %t), want a non-empty generated default", value, ok)
		}
	})
}

func TestFieldWithValueBooleanLiterals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		schema      string
		input       string
		want        bool
		wantError   bool
		wantString  string
		stringValue bool
		secret      bool
	}{
		{name: "true literal", schema: "bool", input: "true", want: true},
		{name: "false literal", schema: "bool", input: "false"},
		{name: "replace true with false", schema: "true", input: "false"},
		{name: "replace false with true", schema: "false", input: "true", want: true},
		{name: "false overrides true default", schema: "*true | bool", input: "false"},
		{name: "format attributes before parsing", schema: "bool @cuebook(trim)", input: " true ", want: true},
		{name: "empty input rejected", schema: "bool", wantError: true},
		{name: "arbitrary input rejected", schema: "bool", input: "invalid", wantError: true},
		{name: "numeric input rejected", schema: "bool", input: "1", wantError: true},
		{name: "uppercase input rejected", schema: "bool", input: "TRUE", wantError: true},
		{name: "quoted boolean rejected", schema: "bool", input: `"false"`, wantError: true},
		{name: "expression rejected", schema: "bool", input: "true | false", wantError: true},
		{name: "string encoding unchanged", schema: "string", input: "false", stringValue: true, wantString: "false"},
		{name: "integer encoding unchanged", schema: "int", input: "12", stringValue: true, wantString: "12"},
		{name: "float encoding unchanged", schema: "number", input: "1.5", stringValue: true, wantString: "1.5"},
		{name: "mixed type encoding unchanged", schema: "bool | string", input: "true", stringValue: true, wantString: "true"},
		{name: "secret boolean retains password formatting", schema: "bool @cuebook(argon2id)", input: "true", stringValue: true, secret: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ctx := cuecontext.New()
			compiled := ctx.CompileString("Value: " + test.schema)
			if err := compiled.Err(); err != nil {
				t.Fatal(err)
			}
			field := NewField("Value", compiled.LookupPath(cue.ParsePath("Value")))
			updated, err := field.WithValue(test.input)
			if test.wantError {
				if err == nil || updated != nil {
					t.Fatalf("WithValue(%q) = %v, err = %v; want nil field and an error", test.input, updated, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}

			value := ctx.BuildExpr(ast.NewStruct(updated)).LookupPath(cue.ParsePath("Value"))
			if test.stringValue {
				got, err := value.String()
				if err != nil {
					t.Fatalf("updated value is not a string: %v", err)
				}
				if test.secret {
					if !strings.HasPrefix(got, "$argon2id$") {
						t.Fatal("secret boolean was not encoded as an Argon2id password string")
					}
				} else if got != test.wantString {
					t.Errorf("updated string = %q, want %q", got, test.wantString)
				}
				return
			}
			if got, err := value.Bool(); err != nil || got != test.want {
				t.Errorf("updated boolean = %t, err = %v; want %t", got, err, test.want)
			}
		})
	}
}

func TestFieldMarshalsByCUEType(t *testing.T) {
	t.Parallel()

	compiled := cuecontext.New().CompileString(`{
		Enabled: false
		Count: 12
		Ratio: 1.5
		Plain: "line one\nline two"
		Notes: "line one\nline two" @cuebook(multiline)
		Items: [1, 2]
		Object: {Name: "Ada"}
		Empty: null
		Secret: "hash" @cuebook(argon2id)
	}`)
	if err := compiled.Err(); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "boolean", path: "Enabled", want: "false"},
		{name: "integer", path: "Count", want: "12"},
		{name: "float", path: "Ratio", want: "1.5"},
		{name: "null", path: "Empty", want: "null"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
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
		name        string
		path        string
		wantNewline bool
	}{
		{name: "single-line string escapes newline", path: "Plain"},
		{name: "multiline string retains newline", path: "Notes", wantNewline: true},
		{name: "password remains a quoted string", path: "Secret"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			field := NewField(test.path, compiled.LookupPath(cue.ParsePath(test.path)))
			got, err := field.MarshalText()
			if err != nil {
				t.Fatal(err)
			}
			containsNewline := strings.Contains(string(got), "\n")
			if containsNewline != test.wantNewline {
				t.Errorf("MarshalText() contains literal newline = %t, want %t: %q", containsNewline, test.wantNewline, got)
			}
			if test.path == "Secret" && string(got) != `"hash"` {
				t.Errorf("MarshalText() = %q, want %q", got, `"hash"`)
			}

			updated, err := field.WithValue("updated one\nupdated two")
			if err != nil {
				t.Fatal(err)
			}
			literal, ok := updated.Value.(*ast.BasicLit)
			if !ok {
				t.Fatalf("updated field value has type %T, want *ast.BasicLit", updated.Value)
			}
			literalHasNewline := strings.Contains(literal.Value, "\n")
			if literalHasNewline != test.wantNewline {
				t.Errorf("WithValue() literal contains newline = %t, want %t: %q", literalHasNewline, test.wantNewline, literal.Value)
			}
		})
	}

	for _, test := range []struct {
		name string
		path string
	}{
		{name: "list JSON", path: "Items"},
		{name: "struct JSON", path: "Object"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := NewField(test.name, compiled.LookupPath(cue.ParsePath(test.path))).MarshallText()
			if err != nil {
				t.Fatal(err)
			}
			if !json.Valid(got) {
				t.Errorf("MarshallText() = %q, want valid JSON", got)
			}
		})
	}
}
