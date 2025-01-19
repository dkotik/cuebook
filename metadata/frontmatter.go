package metadata

import (
	"bytes"

	"github.com/dkotik/cuebook/patch"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

type Frontmatter struct {
	ByteRange patch.ByteRange
	ast.Node
	Source []byte
}

func (m Frontmatter) Title() string {
	if m.Node != nil && m.Node.HasChildren() {
		first := m.Node.FirstChild()
		return string(first.Lines().Value(m.Source))
	}
	return ""
}

func (m Frontmatter) Description() string {
	if m.Node == nil {
		return ""
	}
	if total := m.Node.ChildCount(); total > 1 {
		b := bytes.Buffer{}
		next := m.Node.FirstChild()
		for range total - 1 {
			next = next.NextSibling()
			lines := next.Lines()
			for i := range lines.Len() {
				line := lines.At(i)
				// _, _ = b.Write(m.Source[line.Start:line.Stop])
				_, _ = b.Write(line.Value(m.Source))
				_, _ = b.WriteRune('\n')
			}
		}
		return b.String()
	}
	return ""
}

func (m Frontmatter) Get(frontMatterFieldName string) any {
	return m.Node.OwnerDocument().Meta()[frontMatterFieldName]
	// value, _ := m.Node.OwnerDocument().Meta()[frontMatterFieldName]
	// return value
}

func NewFrontmatter(source []byte) Frontmatter {
	source, tail := ReadLeadingComments(source)
	return Frontmatter{
		ByteRange: patch.ByteRange{
			Head: 0,
			Tail: tail,
		},
		Node:   goldmark.New().Parser().Parse(text.NewReader(source)),
		Source: source,
	}
}
