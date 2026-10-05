package metadata

import (
	"bytes"

	"github.com/yuin/goldmark/v2/ast"
	"github.com/yuin/goldmark/v2/parser"
)

type Frontmatter struct {
	ast.Node
	Source           []byte
	TailBytePosition int
}

func (m Frontmatter) Title() string {
	if m.Node != nil && m.Node.HasChildren() {
		first, ok := m.Node.FirstChild().(ast.BlockNode)
		if !ok {
			return ""
		}

		var title bytes.Buffer
		for i, segment := range first.Source() {
			if i > 0 {
				_ = title.WriteByte('\n')
			}
			value := bytes.TrimSuffix(segment.Bytes(m.Source), []byte("\n"))
			value = bytes.TrimSuffix(value, []byte("\r"))
			_, _ = title.Write(value)
		}
		return title.String()
	}
	return ""
}

func (m Frontmatter) Description() string {
	if m.Node == nil {
		return ""
	}
	if m.Node.ChildCount() > 1 {
		var description bytes.Buffer
		for next := m.Node.FirstChild().NextSibling(); next != nil; next = next.NextSibling() {
			block, ok := next.(ast.BlockNode)
			if !ok {
				continue
			}
			for _, segment := range block.Source() {
				value := bytes.TrimSuffix(segment.Bytes(m.Source), []byte("\n"))
				value = bytes.TrimSuffix(value, []byte("\r"))
				_, _ = description.Write(value)
				_, _ = description.WriteRune('\n')
			}
		}
		return description.String()
	}
	return ""
}

func (m Frontmatter) Get(frontMatterFieldName string) any {
	if m.Node == nil {
		return nil
	}
	return m.Node.OwnerDocument().Metadata()[frontMatterFieldName]
}

func NewFrontmatter(source []byte) Frontmatter {
	source, tail := ReadLeadingComments(source)
	return Frontmatter{
		Node:             parser.New().Parse(source),
		Source:           source,
		TailBytePosition: tail,
	}
}
