package metadata

import (
	"bytes"
	"fmt"

	"cuelang.org/go/cue"
)

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
