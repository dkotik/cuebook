package cuebook

import (
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
