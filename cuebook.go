package cuebook

import "cuelang.org/go/cue"

// Book holds a list of maps.
type Book struct {
	sources []string
}

func NewBook(sources ...string) (*Book, error) {
	return &Book{sources: sources}, nil
}

func GetByteSpanInSource(v cue.Value) (start, end int, ok bool) {
	_, expressions := v.Expr()
	for _, expression := range expressions {
		if expression.IsConcrete() {
			if source := expression.Source(); source != nil {
				// found first concrete data definition
				// with present source pointer
				return source.Pos().Offset(), source.End().Offset(), true
			}
		}
	}
	return 0, 0, false
}
