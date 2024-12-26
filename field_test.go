package cuebook_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/literal"
	"github.com/dkotik/cuebook"
)

const basicStruct = `{
	first: "first",
	// comment
	second: "second",
	// another comment
	third: "third",
}`

func SubstituteValue(w io.Writer, source []byte, v cue.Value, value string) (err error) {
	byteRange := cuebook.GetByteSpanInSource(v)
	if !byteRange.IsValid() {
		return errors.New("unable to determine buffer bounds")
	}
	if _, err = io.Copy(w, bytes.NewReader(source[:byteRange.BeginsAt])); err != nil {
		return err
	}

	if label, ok := v.Label(); ok {
		// TODO: quote label? literal.Label.Quote(label)
		if _, err = io.WriteString(w, label); err != nil {
			return err
		}
		if _, err = w.Write([]byte(`: `)); err != nil {
			return err
		}
	}

	switch kind := v.Kind(); kind {
	case cue.StringKind:
		// TODO: adjust tab indent based on line position
		if _, err = io.WriteString(w, literal.String.WithOptionalTabIndent(2).Quote(value)); err != nil {
			return err
		}
	case cue.BoolKind, cue.FloatKind, cue.BytesKind, cue.ListKind, cue.NumberKind, cue.StructKind:
		// TODO: implement
		fallthrough
	default:
		return fmt.Errorf("unsupport value kind: %s", kind)
	}

	if _, err = io.Copy(w, bytes.NewReader(source[byteRange.EndsAt:])); err != nil {
		return err
	}
	return nil
}

func TestFieldSubstitution(t *testing.T) {
	tree := cuecontext.New().CompileBytes([]byte(basicStruct))
	err := tree.Err()
	if err != nil {
		t.Fatal(err)
	}
	value := tree.LookupPath(cue.ParsePath("second"))
	if err = value.Err(); err != nil {
		t.Fatal(err)
	}
	b := &bytes.Buffer{}
	t.Log(value.Label())
	SubstituteValue(b, []byte(basicStruct), value.Value(), "???\n???")
	// t.Fatal(b.String())
}
