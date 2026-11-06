package cuebook

import (
	"fmt"
	"os"
	"reflect"
	"slices"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
)

func TestEntryGetDescriptionIncludesAllFieldsAndUsesPlaceholders(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   []string
	}{
		{
			name:   "regular and detail fields including title",
			source: `[{ Name: "Ada" @cuebook(title), Email: "ada@example.test", Note: "Additional context" @cuebook(detail) }]`,
			want:   []string{"Ada", "ada@example.test", "Additional context"},
		},
		{
			name:   "empty regular and detail fields use ellipsis",
			source: `[{ Name: "" @cuebook(title), Email: "  ", Note: "" @cuebook(detail), Null: null }]`,
			want:   []string{"...", "...", "...", "..."},
		},
		{
			name: "absent optional regular and detail fields use ellipsis",
			source: `#person: { Name: string, Nick?: string, Note?: string @cuebook(detail) }
			[...#person] & [{Name: "Ada"}]`,
			want: []string{"Ada", "...", "..."},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			book, err := New([]byte(test.source))
			if err != nil {
				t.Fatal(err)
			}
			entry, err := book.GetEntry(0)
			if err != nil {
				t.Fatal(err)
			}
			if got := entry.GetDescription(); !reflect.DeepEqual(got, test.want) {
				t.Errorf("GetDescription() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestBookEntryClassifiesFieldsOutsideDefinitionsAsDetails(t *testing.T) {
	tests := []struct {
		name       string
		source     string
		entryIndex int
		wantFields []string
		wantDetail []string
	}{
		{
			name: "uses abstract list definitions and treats concrete-only optional fields as details",
			source: `#Entry: {
				Name: string
				AbstractOptional?: string
				...
			}
			[...#Entry] & [{Name: "Ada", ConcreteOptional?: string}]`,
			wantFields: []string{"Name", "AbstractOptional"},
			wantDetail: []string{"ConcreteOptional"},
		},
		{
			name: "uses the book definitions for later concrete entries",
			source: `[
				{Name: "Ada"},
				{Name: "Grace", Additional: "notes"},
			]`,
			entryIndex: 1,
			wantFields: []string{"Name"},
			wantDetail: []string{"Additional"},
		},
		{
			name: "keeps explicitly annotated detail fields in details",
			source: `#Entry: {
				Name: string
				Notes?: string @cuebook(detail)
			}
			[...#Entry] & [{Name: "Ada", Notes: "more"}]`,
			wantFields: []string{"Name"},
			wantDetail: []string{"Notes"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			book, err := New([]byte(test.source))
			if err != nil {
				t.Fatal(err)
			}

			entry, err := book.GetEntry(test.entryIndex)
			if err != nil {
				t.Fatal(err)
			}

			gotFields := make([]string, len(entry.Fields))
			for index, field := range entry.Fields {
				gotFields[index] = field.Name
			}
			gotDetails := make([]string, len(entry.Details))
			for index, field := range entry.Details {
				gotDetails[index] = field.Name
			}
			slices.Sort(gotFields)
			slices.Sort(gotDetails)
			wantFields := slices.Clone(test.wantFields)
			wantDetails := slices.Clone(test.wantDetail)
			slices.Sort(wantFields)
			slices.Sort(wantDetails)
			if !slices.Equal(gotFields, wantFields) {
				t.Errorf("regular fields = %v, want %v", gotFields, wantFields)
			}
			if !slices.Equal(gotDetails, wantDetails) {
				t.Errorf("detail fields = %v, want %v", gotDetails, wantDetails)
			}
		})
	}
}

func TestRemainingFieldComposition(t *testing.T) {
	source := []byte(`
		#contact :{
			one: string
			another: string
			...
		}

		[...#contact] & [{
			one: "ok"
			another: "ok"
			two: "ok"
		}]
	`)
	book, err := New(source)
	if err != nil {
		t.Fatal(err)
	}
	value := book.Value.LookupPath(cue.MakePath(cue.Index(0)))

	iterator, err := value.Fields()
	if err != nil {
		t.Fatal(err)
	}
	for iterator.Next() {
		t.Log("field found:", iterator.Selector().String(), iterator.Value().IsConcrete())
	}

	entry, err := book.GetEntry(0)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range entry.Fields {
		t.Log("Discovered:", field.Name, field.Value)
	}
	for _, field := range entry.Details {
		t.Log("Discovered:", field.Name, field.Value)
	}

	// t.Fatal("impl")
}

func requirePresentFieldDefinitions(source []byte, expectedCount int) func(*testing.T) {
	return func(t *testing.T) {
		book := cuecontext.New().CompileBytes(source)
		err := book.Err()
		if err != nil {
			t.Fatal("unable to compile Cue source:", err)
		}
		count := 0
		for definition := range EachFieldDefinition(book) {
			t.Log(definition.Selector.String(), definition.Value)
			count++
		}
		if count != expectedCount {
			t.Fatalf("extracted %d field definitions, but should have found %d instead", count, expectedCount)
		}
	}
}

func TestFieldDefinitionExtraction(t *testing.T) {
	large, err := os.ReadFile("./test/testdata/simple.cue")
	if err != nil {
		t.Fatal("unable to read test file")
	}
	t.Run("simple.cue", requirePresentFieldDefinitions(large, 4))

	for i, testCase := range [...]struct {
		Source              []byte
		ExpectedDefinitions int
	}{
		{
			Source: []byte(`[
		...{
						Name: string | *"default"
						Email: string
						Final: number
						Great: 1
					}
					]&[]`),
			ExpectedDefinitions: 4,
		},
		{
			Source: []byte(`[
		...{
						Name: string | *"default"
						Email: string
					}
					]&[{Name: "one"}]`),
			ExpectedDefinitions: 2,
		},
		{
			Source: []byte(`
				#contact: {
					Name: string | *"default"
					Email: string
				}

				[...#contact]&[]`),
			ExpectedDefinitions: 2,
		},
		{
			Source: []byte(`
				#contact: {
					Name: string | *"default"
					Email: string
				}

				[...#contact]&[{Name: "one"}]`),
			ExpectedDefinitions: 2,
		},
		{
			Source:              []byte(``),
			ExpectedDefinitions: 0,
		},
		{
			Source:              []byte(`{}`),
			ExpectedDefinitions: 0,
		},
		{
			Source: []byte(`
				#contact: {
					Name: string | *"default"
					Email: string
				}`),
			ExpectedDefinitions: 0,
		},
		{
			Source:              []byte(`1`),
			ExpectedDefinitions: 0,
		},
	} {
		t.Run(fmt.Sprintf("testing source %d", i), requirePresentFieldDefinitions(testCase.Source, testCase.ExpectedDefinitions))
	}
}

func TestEachFieldDefinitionUsesAbstractListConjunct(t *testing.T) {
	tests := []struct {
		name      string
		source    string
		want      []string
		kindField string
		wantKind  cue.Kind
	}{
		{
			name: "skips an optional field only in the concrete entry",
			source: `#Entry: {
				Name: string
				AbstractOptional?: string
			}

			[...#Entry] & [{Name: "Ada", ConcreteOptional?: string}]`,
			want: []string{"Name", "AbstractOptional"},
		},
		{
			name: "retains abstract fields when the concrete list is empty",
			source: `#Entry: {
				Name: string
				AbstractOptional?: string
			}

			[...#Entry] & []`,
			want: []string{"Name", "AbstractOptional"},
		},
		{
			name: "unifies multiple abstract list constraints",
			source: `#Entry: {
				Name: string
				AbstractOptional?: string
				...
			}

			[...#Entry] & [...{Additional: string}] & [{Name: "Ada", Additional: "value", ConcreteOptional?: string}]`,
			want: []string{"Name", "AbstractOptional", "Additional"},
		},
		{
			name: "uses the left side of an abstract disjunction",
			source: `[...({
				Name: string
				LeftOptional?: string
			} | {
				Name: string
				RightOptional?: string
			})] & [{Name: "Ada"}]`,
			want: []string{"Name", "LeftOptional"},
		},
		{
			name: "uses the left schema alternative and preserves field alternatives",
			source: `#email: =~"^[^@]+@[^@]+$"
			[...({
				Name: string
				Email: #email | [...#email]
				LeftOptional?: string
			} | {
				Name: string
				Email: #email | [...#email]
				RightOptional?: string
			})] & [{Name: "Ada", Email: "ada@example.test"}]`,
			want:      []string{"Name", "Email", "LeftOptional"},
			kindField: "Email",
			wantKind:  cue.StringKind,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			value := cuecontext.New().CompileString(test.source)
			if err := value.Err(); err != nil {
				t.Fatal(err)
			}

			var got []string
			for definition := range EachFieldDefinition(value) {
				name := definition.Selector.Unquoted()
				got = append(got, name)
				if name == test.kindField && definition.Value.IncompleteKind() != test.wantKind {
					t.Errorf("%s definition kind = %s, want %s", name, definition.Value.IncompleteKind(), test.wantKind)
				}
			}
			slices.Sort(got)
			want := slices.Clone(test.want)
			slices.Sort(want)
			if !slices.Equal(got, want) {
				t.Errorf("EachFieldDefinition() fields = %v, want %v", got, want)
			}
		})
	}
}
