package cuebook

import (
	"testing"
)

func TestFieldSubstitution(t *testing.T) {
	source := []byte(`[{
// top comment
wohoo: "bohoo"
// first comment
first: "first" // sidecomment
// comment
second: "second"
// another comment
third: "third"
}]`)
	book, err := New(source)
	if err != nil {
		t.Fatal(err)
	}

	for entry, err := range book.EachEntry() {
		if err != nil {
			t.Fatal(err)
		}
		patch, err := entry.Fields[0].WithStringValue(source, "???\n???")
		if err != nil {
			t.Fatal(err)
		}
		result, err := patch.Apply(source)
		if err != nil {
			t.Fatal(err)
		}
		t.Log(string(result.Source))
		// t.Fatal(string(f.RemoveFromSource([]byte(basicStruct))))
		// t.Fatal("check result")
	}

}
