package cuebook

import (
	"errors"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
)

func TestSwapFields(t *testing.T) {
	t.Parallel()

	const schema = `#task: {
	Title: string @cuebook(title)
	Owner: string
	Notes?: string @cuebook(detail)
	...
}
`
	tests := []struct {
		name    string
		source  string
		entry   int
		a, b    string
		want    string
		wantErr error
	}{
		{
			name:   "adjacent fields",
			source: `[{a: 1, b: 2}]`,
			a:      "a", b: "b",
			want: `[{b: 2, a: 1}]`,
		},
		{
			name:   "argument order does not matter",
			source: `[{a: 1, b: 2}]`,
			a:      "b", b: "a",
			want: `[{b: 2, a: 1}]`,
		},
		{
			name:   "fields with a field between them",
			source: `[{a: 1, b: 2, c: 3}]`,
			a:      "a", b: "c",
			want: `[{c: 3, b: 2, a: 1}]`,
		},
		{
			name:   "labels and attributes move with values",
			source: schema + "[...#task] & [\n  {\n\t\tTitle: \"Plan\" @cuebook(title)\n\t\tOwner: \"Ada\"\n\t},\n]",
			a:      "Title", b: "Owner",
			want: schema + "[...#task] & [\n  {\n\t\tOwner: \"Ada\"\n\t\tTitle: \"Plan\" @cuebook(title)\n\t},\n]",
		},
		{
			name:   "identical fields in earlier entries stay put",
			source: `[{a: 1, b: 2}, {a: 1, b: 2}]`,
			entry:  1,
			a:      "a", b: "b",
			want: `[{a: 1, b: 2}, {b: 2, a: 1}]`,
		},
		{
			name:   "quoted labels",
			source: `[{"first name": "Ada", age: 36}]`,
			a:      "first name", b: "age",
			want: `[{age: 36, "first name": "Ada"}]`,
		},
		{
			name:   "missing field",
			source: `[{a: 1, b: 2}]`,
			a:      "a", b: "missing",
			wantErr: ErrTargetFieldNotStructField,
		},
		{
			name:   "schema field that is not written in the entry",
			source: schema + `[...#task] & [{Title: "Plan", Owner: "Ada"}]`,
			a:      "Title", b: "Notes",
			wantErr: ErrTargetFieldNotStructField,
		},
		{
			name:   "entry that is not a struct",
			source: `["text", "more"]`,
			a:      "a", b: "b",
			wantErr: ErrEntryNotStruct,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			source := []byte(tt.source)
			document := cuecontext.New().CompileBytes(source)
			if err := document.Err(); err != nil {
				t.Fatal(err)
			}
			entry := document.LookupPath(cue.MakePath(cue.Index(tt.entry)))

			patch, err := SwapFields(source, entry, tt.a, tt.b)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("SwapFields() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			got, err := patch.ApplyToCueSource(source)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Errorf("swapped source =\n%s\nwant\n%s", got, tt.want)
			}
			t.Run("reverse", ensureInversible(source, patch))
		})
	}
}
