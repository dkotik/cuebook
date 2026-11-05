package metadata

import (
	"bytes"
	"fmt"

	"cuelang.org/go/cue"
)

// IsOpen reports whether a struct or list accepts fields or elements beyond
// those it declares, as written with the "..." operator. Plain structs and
// structs with a pattern constraint matching every label, such as
// [string]: int, are open too, because CUE treats "..." as [string]: _.
// Closed structs and definitions without "..." are not open. A disjunction
// is open when any of its structs is open. Values of other kinds, including
// top (_), are never open.
func IsOpen(v cue.Value) bool {
	if !v.Exists() {
		return false
	}
	switch v.IncompleteKind() {
	case cue.StructKind:
		return v.Allows(cue.AnyString)
	case cue.ListKind:
		return v.Allows(cue.AnyIndex)
	default:
		return false
	}
}

func ValueToString(v cue.Value) string {
	switch k := v.Kind(); k {
	case cue.BoolKind:
		value, err := v.Bool()
		if err != nil || !value {
			return "X"
		}
		return "true"
	case cue.IntKind, cue.FloatKind, cue.NumberKind:
		b := &bytes.Buffer{}
		_, _ = fmt.Fprintf(b, "%v", v)
		return b.String()
	case cue.StringKind:
		value, _ := v.String()
		return value
	case cue.BytesKind, cue.ListKind, cue.StructKind:
		value, _ := v.MarshalJSON()
		return string(value)
	case cue.NullKind, cue.BottomKind:
		fallthrough
	default:
		return ""
	}
}
