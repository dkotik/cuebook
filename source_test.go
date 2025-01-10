package cuebook

import (
	"testing"

	"cuelang.org/go/cue/cuecontext"
)

func TestNewEntryInsertion(t *testing.T) {
	source := []byte(`[
		{},
		{}
]`)
	patch, err := NewEntryFromDefinition(source, []FieldDefinition{
		{Name: "test", EncodedValue: "\"string\""},
		{Name: "test2", EncodedValue: "\"string2\""},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(patch.ReplaceWith))
	// t.Fatal("patch")

	result, err := patch.Apply(source)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(result.Source))
	// t.Fatal("worked")
}

func TestIsSame(t *testing.T) {
	t.Parallel()

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

	copy := value

	if !IsSame(value, copy) {
		t.Fatal("expected to be the same")
	}
}
