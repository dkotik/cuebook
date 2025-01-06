package cuebook

import (
	"testing"

	"cuelang.org/go/cue/cuecontext"
)

func TestIsSame(t *testing.T) {
	t.Parallel()

	value := cuecontext.New().CompileBytes([]byte(`
		#contact :{
			one: string
			another: string
			...
		}

	 	#contact & {
			one: "ok"
			another: "ok"
			two: "ok"
		}
	`))
	if err := value.Err(); err != nil {
		t.Fatal(err)
	}

	copy := value

	if !IsSame(value, copy) {
		t.Fatal("expected to be the same")
	}
}
