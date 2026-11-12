// Package markdown centralizes the Goldmark parser and HTML renderer used by
// the project.
package markdown

import (
	"github.com/yuin/goldmark/v2/extension"
	"github.com/yuin/goldmark/v2/parser"
	markdownhtml "github.com/yuin/goldmark/v2/renderer/html"
)

// NewParser creates a Goldmark parser with task-list, table, strikethrough,
// footnote, definition-list, and typography support, along with the provided options.
func NewParser(options ...parser.Option) parser.Parser {
	options = append([]parser.Option{parser.WithExtensions(
		extension.NewTaskListItemParser(),
		extension.NewTableParser(),
		extension.NewStrikethroughParser(),
		extension.NewFootnoteParser(),
		extension.NewDefinitionListParser(),
		extension.NewTypographerParser(),
	)}, options...)
	return parser.New(options...)
}

// NewRenderer creates a Goldmark HTML renderer with support for task-list
// checkboxes, tables, strikethrough, footnotes, and definition lists, along with
// the provided options.
func NewRenderer(options ...markdownhtml.Option) markdownhtml.Renderer {
	options = append([]markdownhtml.Option{
		markdownhtml.WithExtensions(
			extension.NewTaskListItemHTMLRenderer(),
			extension.NewTableHTMLRenderer(),
			extension.NewStrikethroughHTMLRenderer(),
			extension.NewFootnoteHTMLRenderer(),
			extension.NewDefinitionListHTMLRenderer(),
		),
	}, options...)
	return markdownhtml.New(options...)
}
