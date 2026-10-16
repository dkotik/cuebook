package field

import (
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/cuecontext"
)

func TestWithValueBooleanLiterals(t *testing.T) {
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
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := cuecontext.New()
			compiled := ctx.CompileString("Value: " + tt.schema)
			if err := compiled.Err(); err != nil {
				t.Fatal(err)
			}
			field := NewField("Value", compiled.LookupPath(cue.ParsePath("Value")))
			updated, err := field.WithValue(tt.input)
			if tt.wantError {
				if err == nil || updated != nil {
					t.Fatalf("WithValue(%q) = %v, err = %v; want nil field and an error", tt.input, updated, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			value := ctx.BuildExpr(ast.NewStruct(updated)).LookupPath(cue.ParsePath("Value"))
			if tt.stringValue {
				got, err := value.String()
				if err != nil {
					t.Fatalf("updated value is not a string: %v", err)
				}
				if tt.secret {
					if !strings.HasPrefix(got, "$argon2id$") {
						t.Fatal("secret boolean was not encoded as an Argon2id password string")
					}
				} else if got != tt.wantString {
					t.Errorf("updated string = %q, want %q", got, tt.wantString)
				}
				return
			}
			if got, err := value.Bool(); err != nil || got != tt.want {
				t.Errorf("updated boolean = %t, err = %v; want %t", got, err, tt.want)
			}
		})
	}
}
