package patch

import (
	"testing"

	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/literal"
	"cuelang.org/go/cue/token"
	"github.com/dkotik/cuebook"
)

func TestInsertPatch(t *testing.T) {
	source := []byte(`[
		{ one: "ok" },
		{	another: "ok" },
		{	another: "ok" },
		{	two: "ok" },
		{	two: "ok" },
		{	two: "ok" }
]`)
	patch, err := AppendToStructList(source, cuecontext.New().BuildExpr(
		ast.NewStruct(
			&ast.Field{
				Label: ast.NewString("Name"),
				Value: ast.NewString("Someone"),
			},
			&ast.Field{
				Label: ast.NewString("Email"),
				Value: ast.NewString("someEmail@somehost.net"),
			},
			&ast.Field{
				Label: ast.NewString("Notes"),
				Value: ast.NewLit(token.STRING, literal.String.WithOptionalTabIndent(1).Quote("str\n sdflkjsdlfk\n\n skf sldkf sdl;f jlksdjf lksdfj lsdk fjsdlfk jsdlf string sdklfj slkdfj sdjf sdklfj sldkfj sldkfj sldfj sldkf jsdlfk jsdlkf jsdlfk jsdlf kjsalfk jasdflk sjadf lskdjf lskdfj sldkfj slkdfj lsdkjf sldkfj lskdjf slkdjf lskdfj lskdfj dslkf")),
			},
		),
	))
	if err != nil {
		t.Fatal(err)
	}
	result, err := patch.ApplyToCueSource(source)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(result))
	_, err = cuebook.New(result)
	if err != nil {
		t.Fatal(err)
	}
	t.Run("reverse", ensureInversible(source, patch))
	// t.Fatal("check")
}
