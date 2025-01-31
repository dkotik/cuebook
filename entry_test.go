package cuebook

import (
	"fmt"
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
		if len(source) == 0 {
			t.Fatal("cannot parse an empty file")
		}
		book := cuecontext.New().CompileBytes(source)
		err := book.Err()
		if err != nil {
			t.Fatal("unable to compile Cue source:", err)
		}
		definitions, err := EachFieldDefinition(book)
		if err != nil {
			t.Fatal(err)
		}
		count := 0
		for selector, field := range definitions {
			t.Log(selector.String(), field.Value())
			count++
		}
		if count != expectedCount {
			t.Fatalf("extracted %d field definitions, but should have found %d instead", count, expectedCount)
		}
	}
}

func TestFieldDefinitionExtraction(t *testing.T) {
	// large, err := os.ReadFile("./test/testdata/simple.cue")
	// if err != nil {
	// 	t.Fatal("unable to read test file")
	// }
	// t.Run("simple.cue", requirePresentFieldDefinitions(large, 5))

	// t.Fatal("check")

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
		// {
		// 	Source: []byte(`
		// 		#contact: {
		// 			Name: string | *"default"
		// 			Email: string
		// 		}

		// 		[...#contact]&[]`),
		// 	ExpectedDefinitions: 2,
		// },
		// {
		// 	Source: []byte(`
		// 		#contact: {
		// 			Name: string | *"default"
		// 			Email: string
		// 		}

		// 		[...#contact]&[{Name: "one"}]`),
		// 	ExpectedDefinitions: 2,
		// },
	} {
		t.Run(fmt.Sprintf("testing source %d", i), requirePresentFieldDefinitions(testCase.Source, testCase.ExpectedDefinitions))
	}

	// book := cuecontext.New().CompileString()
	// err := book.Err()
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// definitions, err := EachFieldDefinition(book)
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// count := 0
	// for selector, field := range definitions {
	// 	t.Log(selector.String(), field.Value())
	// 	count++
	// }

	// if count != 2 {
	// 	t.Fatalf("extracted %d fields, but should have found 2 definitions", count)
	// }
	// t.Fatal(`check`)

	// book, _ = book.Default()
	// t.Fatal(book.Elem())

	// abstract := book.LookupPath(cue.MakePath(cue.AnyIndex))
	// t.Fatal(abstract.Expr())
	// field, err := abstract.Fields(cue.All())
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// t.Fatal(field.Next())

	// for field.Next() {
	// 	t.Log("-----", field.Label())
	// }

	// t.Fatalf("%+v", field)

	// list, err := book.List()
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// list.Next()
	// t.Fatal(list.Value().Source())

	// _, expr := book.Expr()
	// for _, expression := range expr {
	// 	abstract := expression.LookupPath(cue.MakePath(cue.AnyIndex))
	// 	field, err := abstract.Fields(cue.All())
	// 	if err != nil {
	// 		t.Fatal(err)
	// 	}

	// 	for field.Next() {
	// 		t.Log("-----", field.Label())
	// 	}

	// 	// t.Fatal(field.Next())
	// }
}
