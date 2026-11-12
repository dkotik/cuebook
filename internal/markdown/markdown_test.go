package markdown_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/dkotik/cuebook/internal/markdown"
)

func TestNewParserAndRenderer(t *testing.T) {
	t.Parallel()

	source := []byte(`# Hello

A *Markdown* paragraph.

- [ ] Pending
- [x] Done

| Name | Value |
| --- | --- |
| Ada | 1 |

~~Removed~~

Footnote reference[^note].

[^note]: Footnote details.

Term
: Definition details.

"Hello" -- world...`)
	document := markdown.NewParser().Parse(source)

	var rendered bytes.Buffer
	if err := markdown.NewRenderer().Render(&rendered, source, document); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name     string
		contains []string
	}{
		{name: "common Markdown", contains: []string{"<h1>Hello</h1>", "<em>Markdown</em>"}},
		{name: "task lists", contains: []string{`<input disabled="" type="checkbox">`, `<input checked="" disabled="" type="checkbox">`}},
		{name: "tables", contains: []string{"<table>", "<th>Name</th>", "<td>Ada</td>"}},
		{name: "strikethrough", contains: []string{"<del>Removed</del>"}},
		{name: "footnotes", contains: []string{`class="footnotes"`, "Footnote details."}},
		{name: "definition lists", contains: []string{"<dl>", "<dt>Term</dt>", "<dd><p>Definition details.</p>"}},
		{name: "typography", contains: []string{"“Hello”", "–", "…"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			for _, want := range tt.contains {
				if !strings.Contains(rendered.String(), want) {
					t.Errorf("rendered Markdown %q does not contain %q", rendered.String(), want)
				}
			}
		})
	}
}
