package metadata_test

import (
	"reflect"
	"testing"

	"github.com/dkotik/cuebook/metadata"
	"github.com/yuin/goldmark/v2/ast"
	"github.com/yuin/goldmark/v2/parser"
	"github.com/yuin/goldmark/v2/util"
)

func TestDetailsTransformer(t *testing.T) {
	tests := []struct {
		name       string
		source     string
		head       string
		tail       string
		wantBlocks []string
	}{
		{
			name:       "replace first thematic break and append tail after all content",
			source:     "Before\n\n---\n\nMiddle\n\n***\n\nAfter",
			head:       "<details>",
			tail:       "</details>",
			wantBlocks: []string{"paragraph", "html:<details>", "paragraph", "thematic break", "paragraph", "html:</details>"},
		},
		{
			name:       "leave document without thematic break unchanged",
			source:     "First\n\nSecond",
			head:       "<details>",
			tail:       "</details>",
			wantBlocks: []string{"paragraph", "paragraph"},
		},
		{
			name:       "replace thematic break at document start",
			source:     "---\n\nContent",
			head:       "<details>",
			tail:       "</details>",
			wantBlocks: []string{"html:<details>", "paragraph", "html:</details>"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			document, ok := parser.New(parser.WithASTTransformers(util.Prioritized(
				metadata.NewDetailsTransformer(tt.head, tt.tail), 0,
			))).Parse([]byte(tt.source)).(*ast.Document)
			if !ok {
				t.Fatal("Parse() did not return an AST document")
			}

			var gotBlocks []string
			for node := document.FirstChild(); node != nil; node = node.NextSibling() {
				switch node := node.(type) {
				case *ast.Paragraph:
					gotBlocks = append(gotBlocks, "paragraph")
				case *ast.ThematicBreak:
					gotBlocks = append(gotBlocks, "thematic break")
				case *ast.HTMLBlock:
					gotBlocks = append(gotBlocks, "html:"+node.Value.Str(nil))
				default:
					t.Errorf("unexpected top-level node %T", node)
				}
			}

			if !reflect.DeepEqual(gotBlocks, tt.wantBlocks) {
				t.Errorf("document blocks = %#v, want %#v", gotBlocks, tt.wantBlocks)
			}
		})
	}
}
