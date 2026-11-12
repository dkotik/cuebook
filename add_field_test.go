package cuebook

import (
	"errors"
	"testing"
)

func TestAddField(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		source string
		entry  int
		field  string
		kind   FieldKind
		value  string
		want   string
	}{
		{
			name:   "string on a one-line entry",
			source: `[{Name: "Ada"}]`,
			field:  "Role", kind: FieldKindString, value: "dev",
			want: `[{Name: "Ada", Role: "dev"}]`,
		},
		{
			name:   "string escapes quotes and newlines",
			source: `[{Name: "Ada"}]`,
			field:  "Quote", kind: FieldKindString, value: "say \"hi\"\nbye",
			want: `[{Name: "Ada", Quote: "say \"hi\"\nbye"}]`,
		},
		{
			name:   "empty string",
			source: `[{Name: "Ada"}]`,
			field:  "Role", kind: FieldKindString,
			want: `[{Name: "Ada", Role: ""}]`,
		},
		{
			name:   "only the selected entry changes",
			source: `[{Name: "Ada"}, {Name: "Ada"}]`,
			entry:  1, field: "Role", kind: FieldKindString, value: "dev",
			want: `[{Name: "Ada"}, {Name: "Ada", Role: "dev"}]`,
		},
		{
			name:   "multiline string spreads a one-line entry",
			source: `[{Name: "Ada"}]`,
			field:  "Bio", kind: FieldKindMultiLine, value: "line one\nline two",
			want: "[{\n\t\tName: \"Ada\"\n\t\tBio: \"\"\"\n\t\t\tline one\n\t\t\tline two\n\t\t\t\"\"\" @cuebook(multiline)\n\t}]",
		},
		{
			name:   "multiline string normalizes CRLF line endings",
			source: `[{Name: "Ada"}]`,
			field:  "Bio", kind: FieldKindMultiLine, value: "line one\r\nline two",
			want: "[{\n\t\tName: \"Ada\"\n\t\tBio: \"\"\"\n\t\t\tline one\n\t\t\tline two\n\t\t\t\"\"\" @cuebook(multiline)\n\t}]",
		},
		{
			name:   "empty multiline string",
			source: `[{Name: "Ada"}]`,
			field:  "Bio", kind: FieldKindMultiLine,
			want: "[{\n\t\tName: \"Ada\"\n\t\tBio:  \"\" @cuebook(multiline)\n\t}]",
		},
		{
			name:   "multiline entry keeps comments and attributes",
			source: "[\n\t{\n\t\t// leading comment\n\t\tName: \"Ada\" // trailing\n\t\tNote: \"x\" @cuebook(detail)\n\t},\n]",
			field:  "Bio", kind: FieldKindMultiLine, value: "one\ntwo",
			want: "[\n\t{\n\t\t// leading comment\n\t\tName: \"Ada\" // trailing\n\t\tNote: \"x\"   @cuebook(detail)\n\t\tBio: \"\"\"\n\t\t\tone\n\t\t\ttwo\n\t\t\t\"\"\" @cuebook(multiline)\n\t},\n]",
		},
		{
			name:   "integer",
			source: `[{Name: "Ada"}]`,
			field:  "Age", kind: FieldKindNumber, value: " 36 ",
			want: `[{Name: "Ada", Age: 36}]`,
		},
		{
			name:   "negative float",
			source: `[{Name: "Ada"}]`,
			field:  "Balance", kind: FieldKindNumber, value: "-3.5",
			want: `[{Name: "Ada", Balance: -3.5}]`,
		},
		{
			name:   "number literal forms",
			source: `[{Name: "Ada"}]`,
			field:  "Mask", kind: FieldKindNumber, value: "0x1F",
			want: `[{Name: "Ada", Mask: 0x1F}]`,
		},
		{
			name:   "empty number is zero",
			source: `[{Name: "Ada"}]`,
			field:  "Age", kind: FieldKindNumber,
			want: `[{Name: "Ada", Age: 0}]`,
		},
		{
			name:   "boolean true",
			source: "[\n\t{\n\t\tName: \"Ada\"\n\t},\n]",
			field:  "Active", kind: FieldKindBoolean, value: "true",
			want: "[\n\t{\n\t\tName:   \"Ada\"\n\t\tActive: true\n\t},\n]",
		},
		{
			name:   "empty boolean is false",
			source: `[{Name: "Ada"}]`,
			field:  "Active", kind: FieldKindBoolean,
			want: `[{Name: "Ada", Active: false}]`,
		},
		{
			name:   "name that is not an identifier is quoted",
			source: `[{Name: "Ada"}]`,
			field:  "has space", kind: FieldKindString, value: "x",
			want: `[{Name: "Ada", "has space": "x"}]`,
		},
		{
			name:   "hidden-looking name stays a regular field",
			source: `[{Name: "Ada"}]`,
			field:  "_note", kind: FieldKindString, value: "x",
			want: `[{Name: "Ada", "_note": "x"}]`,
		},
		{
			name:   "definition-looking name stays a regular field",
			source: `[{Name: "Ada"}]`,
			field:  "#note", kind: FieldKindString, value: "x",
			want: `[{Name: "Ada", "#note": "x"}]`,
		},
		{
			name:   "empty entry",
			source: `[{}]`,
			field:  "Name", kind: FieldKindString, value: "Ada",
			want: `[{Name: "Ada"}]`,
		},
		{
			name:   "open schema",
			source: "#person: {Name: string, ...}\n[...#person] & [{Name: \"Ada\"}]",
			field:  "Role", kind: FieldKindString, value: "dev",
			want: "#person: {Name: string, ...}\n[...#person] & [{Name: \"Ada\", Role: \"dev\"}]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			source := []byte(tt.source)
			book, err := New(source)
			if err != nil {
				t.Fatal(err)
			}
			entry, err := book.GetEntry(tt.entry)
			if err != nil {
				t.Fatal(err)
			}
			change, err := entry.AddField(tt.field, tt.kind, tt.value)
			if err != nil {
				t.Fatal(err)
			}
			updated, err := change.ApplyToCueSource(source)
			if err != nil {
				t.Fatal(err)
			}
			if string(updated) != tt.want {
				t.Fatalf("source =\n%s\nwant\n%s", updated, tt.want)
			}
			if _, err := New(updated); err != nil {
				t.Fatalf("invalid result: %v", err)
			}
			t.Run("reverse", ensureInversible(source, change))
		})
	}
}

func TestAddFieldRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		source string
		field  string
		kind   FieldKind
		value  string
		want   error
	}{
		{name: "empty name", source: `[{Name: "Ada"}]`, field: "", kind: FieldKindString, want: ErrFieldNameEmpty},
		{name: "blank name", source: `[{Name: "Ada"}]`, field: " \t", kind: FieldKindString, want: ErrFieldNameEmpty},
		{name: "existing field", source: `[{Name: "Ada"}]`, field: "Name", kind: FieldKindString, want: ErrFieldExists},
		{name: "unset optional schema field", source: "#person: {Name: string, Role?: string, ...}\n[...#person] & [{Name: \"Ada\"}]", field: "Role", kind: FieldKindString, want: ErrFieldExists},
		{name: "unknown kind", source: `[{Name: "Ada"}]`, field: "Role", kind: "list", want: ErrUnknownFieldKind},
		{name: "number that is text", source: `[{Name: "Ada"}]`, field: "Age", kind: FieldKindNumber, value: "abc", want: ErrInvalidFieldValue},
		{name: "number expression", source: `[{Name: "Ada"}]`, field: "Age", kind: FieldKindNumber, value: "1 + 2", want: ErrInvalidFieldValue},
		{name: "quoted number", source: `[{Name: "Ada"}]`, field: "Age", kind: FieldKindNumber, value: `"3"`, want: ErrInvalidFieldValue},
		{name: "boolean that is not true or false", source: `[{Name: "Ada"}]`, field: "Active", kind: FieldKindBoolean, value: "yes", want: ErrInvalidFieldValue},
		{name: "entry that is not a struct", source: `["Ada"]`, field: "Role", kind: FieldKindString, want: ErrEntryNotStruct},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			source := []byte(tt.source)
			book, err := New(source)
			if err != nil {
				t.Fatal(err)
			}
			entry, entryErr := book.GetEntry(0)
			if entryErr != nil && !errors.Is(entryErr, ErrEntryValueNotStruct) {
				t.Fatal(entryErr)
			}
			change, err := entry.AddField(tt.field, tt.kind, tt.value)
			if !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
			if change != nil {
				t.Fatalf("change = %v, want nil", change)
			}
		})
	}
}

func TestFieldKindValidateValue(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		kind  FieldKind
		value string
		want  error
	}{
		{name: "any string", kind: FieldKindString, value: "anything \" goes"},
		{name: "any multiline string", kind: FieldKindMultiLine, value: "one\ntwo"},
		{name: "number", kind: FieldKindNumber, value: "-1.5e3"},
		{name: "empty number", kind: FieldKindNumber},
		{name: "text as number", kind: FieldKindNumber, value: "ten", want: ErrInvalidFieldValue},
		{name: "boolean", kind: FieldKindBoolean, value: "true"},
		{name: "empty boolean", kind: FieldKindBoolean},
		{name: "capitalized boolean", kind: FieldKindBoolean, value: "True", want: ErrInvalidFieldValue},
		{name: "unknown kind", kind: "list", want: ErrUnknownFieldKind},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := tt.kind.ValidateValue(tt.value); !errors.Is(err, tt.want) {
				t.Fatalf("ValidateValue() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestAddFieldToClosedSchemaFailsValidation(t *testing.T) {
	t.Parallel()
	source := []byte("#person: {Name: string}\n[...#person] & [{Name: \"Ada\"}]")
	book, err := New(source)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := book.GetEntry(0)
	if err != nil {
		t.Fatal(err)
	}
	change, err := entry.AddField("Role", FieldKindString, "dev")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Validated(change).ApplyToCueSource(source); err == nil {
		t.Fatal("validated patch accepted a field that the closed schema forbids")
	}
}
