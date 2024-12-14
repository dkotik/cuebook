package test

import (
	"fmt"
	"os"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/literal"
	"cuelang.org/go/cue/token"
)

// taken from https://github.com/cue-lang/cue/blob/master/internal/internal.go
func toFile(n ast.Node) *ast.File {
	if n == nil {
		return nil
	}
	switch n := n.(type) {
	case *ast.StructLit:
		f := &ast.File{Decls: n.Elts}
		// Ensure that the comments attached to the struct literal are not lost.
		ast.SetComments(f, ast.Comments(n))
		return f
	case ast.Expr:
		ast.SetRelPos(n, token.NoSpace)
		return &ast.File{Decls: []ast.Decl{&ast.EmbedDecl{Expr: n}}}
	case *ast.File:
		return n
	default:
		panic(fmt.Sprintf("Unsupported node type %T", n))
	}
}

func TestNodeCreation(t *testing.T) {
	source, err := os.ReadFile("testdata/simple.cue")
	if err != nil {
		t.Fatal(err)
	}
	ctx := cuecontext.New()
	tree := ctx.CompileBytes(source)
	if err = tree.Err(); err != nil {
		t.Fatal(err)
	}

	val := ctx.BuildExpr(
		// ast.NewList(
		ast.NewStruct(
			&ast.Field{
				Label: ast.NewString("Name"),
				Value: ast.NewIdent("string"),
				// Value: ast.NewString("something"),
			},
		),
	)

	// &BasicLit{Kind: token.STRING, ValuePos: token.NoPos, Value: str}
	// strConstructor := func(v string) cue.Value {
	// 	return ast.NewString(v)
	// }

	val = val.FillPath(cue.ParsePath("Name"), ast.NewString(literal.String.WithOptionalTabIndent(1).Quote("str\n sdflkjsdlfk\n\n skf sldkf sdl;f jlksdjf lksdfj lsdk fjsdlfk jsdlf string sdklfj slkdfj sdjf sdklfj sldkfj sldkfj sldfj sldkf jsdlfk jsdlkf jsdlfk jsdlf kjsalfk jasdflk sjadf lskdjf lskdfj sldkfj slkdfj lsdkjf sldkfj lskdjf slkdjf lskdfj lskdfj dslkf")))

	// val, err =
	// if err != nil {
	// 	t.Fatal(err)
	// }
	b, err := format.Node(val.Syntax(), format.Simplify())
	if err != nil {
		t.Fatal(b)
	}
	// b, err = format.Source(b)
	// if err != nil {
	// 	t.Fatal(b)
	// }
	// t.Fatal(string(b))
}
