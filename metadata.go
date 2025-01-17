package cuebook

import (
	"bytes"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

type Metadata struct {
	ast.Node
	Source []byte
}

func (m Metadata) Title() string {
	if m.Node != nil && m.Node.HasChildren() {
		first := m.Node.FirstChild()
		return string(first.Lines().Value(m.Source))
	}
	return ""
}

func (m Metadata) Description() string {
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

func (m Metadata) Get(frontMatterFieldName string) any {
	return m.Node.OwnerDocument().Meta()[frontMatterFieldName]
	// value, _ := m.Node.OwnerDocument().Meta()[frontMatterFieldName]
	// return value
}

func (d Document) Metadata() Metadata {
	for _, comment := range d.Doc() {
		source := []byte(comment.Text())
		return Metadata{
			Node:   goldmark.New().Parser().Parse(text.NewReader(source)),
			Source: source,
		}
		// return comment.Text()
	}
	return Metadata{}
}
