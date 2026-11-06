package cuebook

import (
	"errors"
	"reflect"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
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
			if len(book.Definitions) != len(tt.fieldNames) {
				t.Fatalf("got %d field definitions, want %d", len(book.Definitions), len(tt.fieldNames))
			}
			for i, definition := range book.Definitions {
				if name := definition.Selector.Unquoted(); name != tt.fieldNames[i] {
					t.Errorf("definition %d selector = %q, want %q", i, name, tt.fieldNames[i])
				}
				if definition.Value.Kind() != tt.fieldKinds[i] {
					t.Errorf("definition %q kind = %s, want %s", definition.Selector, definition.Value.Kind(), tt.fieldKinds[i])
				}
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
		name        string
		source      string
		wantFields  [][]string
		wantDetails [][]string
	}{
		{
			name: "known fields follow book order and unknown fields become details",
			source: `[
				{ first: "book first", second: "book second" },
				{ unlistedZ: "z", second: "entry second", unlistedA: "a", first: "entry first" },
			]`,
			wantFields: [][]string{
				{"first", "second"},
				{"first", "second"},
			},
			wantDetails: [][]string{
				{},
				{"unlistedZ", "unlistedA"},
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

			var gotFields, gotDetails [][]string
			for entry, err := range book.EachEntry() {
				if err != nil {
					t.Fatal(err)
				}
				fieldNames := make([]string, len(entry.Fields))
				for i, field := range entry.Fields {
					fieldNames[i] = field.Name
				}
				detailNames := make([]string, len(entry.Details))
				for i, field := range entry.Details {
					detailNames[i] = field.Name
				}
				gotFields = append(gotFields, fieldNames)
				gotDetails = append(gotDetails, detailNames)
			}
			if !reflect.DeepEqual(gotFields, tt.wantFields) {
				t.Errorf("entry field order = %#v, want %#v", gotFields, tt.wantFields)
			}
			if !reflect.DeepEqual(gotDetails, tt.wantDetails) {
				t.Errorf("entry details = %#v, want %#v", gotDetails, tt.wantDetails)
			}
		})
	}
}

func TestCuebookErrorCodes(t *testing.T) {
	tests := []struct {
		name string
		code Error
		want string
	}{
		{name: "unknown", code: ErrUnknownCuebookError, want: "unknown cuebook error code #0"},
		{name: "expected list", code: ErrNotCueList, want: "expected a list structued data"},
		{name: "abstract entry", code: ErrCannotLoadAbstractEntry, want: "cannot load an abstract value as structured object"},
		{name: "entry not struct", code: ErrEntryValueNotStruct, want: "value is not a structured object"},
		{name: "entry index", code: ErrBookEntryIndexOutOfRange, want: "not found: index out of range"},
		{name: "field index", code: ErrFieldIndexOutOfRange, want: "not found: index out of range"},
	}
	localizer := i18n.NewLocalizer(i18n.NewBundle(language.AmericanEnglish))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.code.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
			got, err := tt.code.LocalizeWith(localizer)
			if err != nil {
				t.Fatalf("LocalizeWith() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("LocalizeWith() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPatchErrorByteRangeNotFound(t *testing.T) {
	t.Parallel()

	const want = "byte range not found"
	if got := ErrByteRangeNotFound.Error(); got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}

	localizer := i18n.NewLocalizer(i18n.NewBundle(language.AmericanEnglish))
	got, err := ErrByteRangeNotFound.LocalizeWith(localizer)
	if err != nil {
		t.Fatalf("LocalizeWith() error = %v", err)
	}
	if got != want {
		t.Errorf("LocalizeWith() = %q, want %q", got, want)
	}
}

func TestBookOperationsReturnModeledErrors(t *testing.T) {
	tests := []struct {
		name   string
		source []byte
		want   error
	}{
		{name: "root is not a list", source: []byte("{}"), want: ErrNotCueList},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := New(tt.source)
			if !errors.Is(err, tt.want) {
				t.Fatalf("New() error = %v, want errors.Is(error, %v)", err, tt.want)
			}
		})
	}

	if _, err := New([]byte("[")); err == nil {
		t.Fatal("New() error = nil, want a parse error")
	}
	if _, err := New([]byte("[string]")); err == nil {
		t.Fatal("New() error = nil, want a validation error")
	}

	book, err := New([]byte("[]"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := book.GetEntry(-1); !errors.Is(err, ErrBookEntryIndexOutOfRange) {
		t.Errorf("GetEntry(-1) error = %v, want entry index error", err)
	}
	if _, err := book.GetEntry(0); !errors.Is(err, ErrBookEntryIndexOutOfRange) {
		t.Errorf("GetEntry(0) error = %v, want entry index error", err)
	}
	if _, err := book.GetValue(0); err == nil {
		t.Error("GetValue(0) error = nil, want an error for an out-of-range entry")
	}

	value := cuecontext.New().CompileString(`["value"]`)
	invalidEntryBook := Book{Value: value}
	sawEntry := false
	for _, err := range invalidEntryBook.EachEntry() {
		sawEntry = true
		if !errors.Is(err, ErrEntryValueNotStruct) {
			t.Errorf("EachEntry() error = %v, want entry kind error", err)
		}
		break
	}
	if !sawEntry {
		t.Fatal("EachEntry() yielded no entry to report the invalid kind")
	}
}

func TestBookLenReturnsErrorForOpenList(t *testing.T) {
	t.Parallel()

	value := cuecontext.New().CompileString(`[...string]`)
	if _, err := (Book{Value: value}).Len(); err == nil {
		t.Fatal("Len() error = nil, want an error for an open list")
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

	values, err := book.Value.List()
	if err != nil {
		t.Fatal(err)
	}
	var entries []string
	for values.Next() {
		entry := values.Value()
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
