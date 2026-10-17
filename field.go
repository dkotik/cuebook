package cuebook

import (
	"fmt"
	"iter"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
)

type Field struct {
	Name        string
	Description string
	Kind        cue.Kind
	Value       cue.Value
}

// NewField creates a field from a CUE value and its preceding documentation comments.
func NewField(name string, value cue.Value) Field {
	var description strings.Builder
	for _, comment := range value.Doc() {
		text := strings.TrimSpace(comment.Text())
		if text == "" {
			continue
		}
		if description.Len() > 0 {
			description.WriteByte('\n')
		}
		description.WriteString(text)
	}
	return Field{
		Name:        name,
		Description: description.String(),
		Kind:        value.Kind(),
		Value:       value,
	}
}

func EachField(value cue.Value, options ...cue.Option) iter.Seq2[cue.Selector, cue.Value] {
	next, err := value.Fields(options...)
	if err != nil {
		panic(fmt.Errorf("unable to iterate over the Cue list: %w", err))
	}

	return func(yield func(cue.Selector, cue.Value) bool) {
		for next.Next() {
			if !yield(next.Selector(), next.Value()) {
				break
			}
		}
	}
}

func firstDisjunct(value cue.Value) cue.Value {
	if op, disjuncts := value.Expr(); op == cue.OrOp && len(disjuncts) > 0 {
		return disjuncts[0]
	}
	return value
}

// EachFieldDefinition emits abstract field definitions for structured entries
// of a Cue list.
func EachFieldDefinition(value cue.Value) iter.Seq2[cue.Selector, cue.Value] {
	return func(yield func(_ cue.Selector, value cue.Value) bool) {
		path := cue.MakePath(cue.Index(0))
		if length, _ := value.Len().Uint64(); length == 0 {
			// insert an empty entry, so that field definitions are inheritted
			// from conjuctions; otherwise field query will return cue.BottomKind
			// without any way of enumerating the fields
			value = value.FillPath(path, ast.NewStruct())
		}

		entry := firstDisjunct(value.LookupPath(path))
		next, _ := entry.Fields(cue.All())
		for next.Next() {
			if !yield(next.Selector(), next.Value()) {
				break
			}
		}
	}
}
