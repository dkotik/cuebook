package test

import (
	"os"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
)

func TestNodeStructure(t *testing.T) {
	b, err := os.ReadFile("testdata/simple.cue")
	if err != nil {
		t.Fatal(err)
	}
	ctx := cuecontext.New()
	tree := ctx.CompileBytes(b)
	if err = tree.Err(); err != nil {
		t.Fatal(err)
	}

	if err = tree.Validate(); err != nil {
		t.Fatal(err)
	}

	if tree.Kind() != cue.ListKind {
		t.Fatal("not list kind:", tree.Kind())
	}

	element, err := tree.List()
	if err != nil {
		t.Fatal("not a list:", err)
	}

	for element.Next() {
		if element.Value().Kind() != cue.StructKind {
			t.Fatal("not struct kind:", tree.Kind())
		}
		t.Log("struct begins at:", element.Value().Pos())
		// modified := element.Value().(cue.ParsePath("Name"), "????")
		// t.Fatal(modified.Validate())
		// if field.Value().Kind() == cue.StringKind {
		// 	original, _ := field.Value().String()
		// 	newValue := field.Value().Fill(original + "!")
		//  use FillPath!
		// }
		_, exprs := element.Value().Expr()
		for _, v := range exprs {
			if v.IsConcrete() || true {
				// first concrete expression source pos and end
				t.Log("boundaries:", v.Source().Pos().Offset(), v.Source().End().Offset())
			}
		}

		field, err := element.Value().Fields(cue.Optional(true))
		if err != nil {
			t.Fatal("not a struct:", err)
		}
		for field.Next() {
			t.Log(field.Value().Path().String())
			// field.Value().Pos().Offset() - where begins by byte count
			t.Log(field.Value().Pos().Offset(), field.Value().Kind(), field.Value().IsConcrete())
			t.Log(field.Selector().String(), ":", field.Value())
			// value, err := field.Value().Bytes()
			// if err != nil {
			// 	t.Fatal(err)
			// }
			// t.Log(string(value))
		}
		// t.Fatal(root.Value().Kind())
	}
	// t.Fatal("comp")
}
