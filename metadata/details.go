package metadata

import (
	"github.com/yuin/goldmark/v2/ast"
	"github.com/yuin/goldmark/v2/parser"
	"github.com/yuin/goldmark/v2/text"
)

type detailsTransformer struct {
	Head string
	Tail string
}

var _ parser.ASTTransformer = (*detailsTransformer)(nil)

func NewDetailsTransformer(head, tail string) parser.ASTTransformer {
	return &detailsTransformer{Head: head, Tail: tail}
}

func (t *detailsTransformer) Transform(document *ast.Document, _ text.Reader, _ parser.Context) {
	for child := document.FirstChild(); child != nil; child = child.NextSibling() {
		if _, ok := child.(*ast.ThematicBreak); !ok {
			continue
		}

		head := ast.NewHTMLBlock(ast.HTMLBlockKind1)
		head.Value = text.NewLinesFromString(t.Head)
		document.ReplaceChild(child, head)

		tail := ast.NewHTMLBlock(ast.HTMLBlockKind1)
		tail.Value = text.NewLinesFromString(t.Tail)
		document.AppendChild(tail)
		return
	}
}
