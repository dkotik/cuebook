package cuebook

import (
	"reflect"
	"testing"

	"cuelang.org/go/cue"
)

func TestNewPopulatesFieldsFromFirstDisjunct(t *testing.T) {
	tests := []struct {
		name              string
		source            string
		fieldNames        []string
		fieldKinds        []cue.Kind
		fieldJSON         []string
		fieldDescriptions []string
	}{
		{
			name: "first disjunct supplies fields",
			source: `[*{
				// First field description.
				first: "one"
				count: 1
			} | { second: true }]`,
			fieldNames:        []string{"first", "count"},
			fieldKinds:        []cue.Kind{cue.StringKind, cue.IntKind},
			fieldJSON:         []string{`"one"`, "1"},
			fieldDescriptions: []string{"First field description.", ""},
		},
		{
			name:              "ordinary list uses first entry",
			source:            `[{ first: "one" }, { ignored: true }]`,
			fieldNames:        []string{"first"},
			fieldKinds:        []cue.Kind{cue.StringKind},
			fieldJSON:         []string{`"one"`},
			fieldDescriptions: []string{""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			book, err := New([]byte(tt.source))
			if err != nil {
				t.Fatal(err)
			}
			if len(book.Fields) != len(tt.fieldNames) {
				t.Fatalf("got %d fields, want %d", len(book.Fields), len(tt.fieldNames))
			}
			for i, field := range book.Fields {
				if field.Name != tt.fieldNames[i] {
					t.Errorf("field %d name = %q, want %q", i, field.Name, tt.fieldNames[i])
				}
				if field.Kind != tt.fieldKinds[i] {
					t.Errorf("field %q kind = %s, want %s", field.Name, field.Kind, tt.fieldKinds[i])
				}
				if field.Description != tt.fieldDescriptions[i] {
					t.Errorf("field %q description = %q, want %q", field.Name, field.Description, tt.fieldDescriptions[i])
				}
				gotJSON, err := field.Value.MarshalJSON()
				if err != nil {
					t.Fatalf("marshal field %q: %v", field.Name, err)
				}
				if string(gotJSON) != tt.fieldJSON[i] {
					t.Errorf("field %q value = %s, want %s", field.Name, gotJSON, tt.fieldJSON[i])
				}
			}
		})
	}
}

func TestEachEntryOrdersFieldsByBookFields(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   [][]string
	}{
		{
			name: "known fields follow book order and unknown fields stay stable",
			source: `[
				{ first: "book first", second: "book second" },
				{ unlistedZ: "z", second: "entry second", unlistedA: "a", first: "entry first" },
			]`,
			want: [][]string{
				{"first", "second"},
				{"first", "second", "unlistedZ", "unlistedA"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			book, err := New([]byte(tt.source))
			if err != nil {
				t.Fatal(err)
			}

			var got [][]string
			for entry, err := range book.EachEntry() {
				if err != nil {
					t.Fatal(err)
				}
				fieldNames := make([]string, len(entry.Fields))
				for i, field := range entry.Fields {
					fieldNames[i] = field.Name
				}
				got = append(got, fieldNames)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("entry field order = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestValueIterator(t *testing.T) {
	book, err := New([]byte(`[
		{ "key": "value" },
		{ "key2": "value2" }
	]`))
	if err != nil {
		t.Fatal(err)
	}

	var entries []string
	for entry := range EachValue(book.Value) {
		b, err := entry.MarshalJSON()
		if err != nil {
			t.Fatal(err)
		}
		entries = append(entries, string(b))
	}

	if len(entries) != 2 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0] != `{"key":"value"}` {
		t.Fatalf(`expected entry to be {"key":"value"}, got %s`, entries[0])
	}
	if entries[1] != `{"key2":"value2"}` {
		t.Fatalf(`expected entry to be {"key2":"value2"}, got %s`, entries[0])
	}
}
