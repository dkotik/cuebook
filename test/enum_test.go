package test

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
)

func TestEnumeratedStringsWithDefault(t *testing.T) {
	value := cuecontext.New().CompileBytes([]byte(`
		#contact :{
			field?: "one" | *"two" | "three" | null
		}

	 	#contact & {
			field: null
		}
	`))
	if err := value.Err(); err != nil {
		t.Fatal(err)
	}

	field := value.LookupPath(cue.ParsePath("field")) //.Fields(cue.All())
	if err := field.Err(); err != nil {
		t.Fatal(err)
	}

	op, values := field.Expr()
	t.Log("operation for field itself:", op.String())
	for _, value := range values[:1] { // first
		op2, options := value.Expr()
		t.Log("operation for firs set of possible enumerations:", op2.String())
		if op2 == cue.NoOp {
			t.Fatal("defined without enumeration")
		}
		if op2 != cue.OrOp {
			t.Fatal("not an | operation")
		}
		for _, option := range options {
			// t.Log(option.Default())
			t.Log(option)
		}
		t.Log(value.Default()) // <- default is discovered here
	}

	// for fields.Next() {
	// 	t.Log(fields.Selector().String())
	// }

	// t.Fatal("impl")
}
