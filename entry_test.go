package cuebook

import (
	"fmt"
	"os"
	"testing"

	"cuelang.org/go/cue/cuecontext"
)

func TestRemainingFieldComposition(t *testing.T) {
	value := cuecontext.New().CompileBytes([]byte(`
		#contact :{
			one: string
			another: string
			...
		}

	 	#contact & {
			one: "ok"
			another: "ok"
			two: "ok"
		}
	`))
	if err := value.Err(); err != nil {
		t.Fatal(err)
	}

	iterator, err := value.Fields()
	if err != nil {
		t.Fatal(err)
	}
	for iterator.Next() {
		t.Log("field found:", iterator.Selector().String(), iterator.Value().IsConcrete())
	}

	entry, err := NewEntry(value)
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
		for selector, field := range EachFieldDefinition(book) {
			t.Log(selector.String(), field.Value())
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
