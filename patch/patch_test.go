package patch

import (
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/cuecontext"
)

func TestPatchApplication(t *testing.T) {
	source := []byte(`[
{ one: "ok" },
{	another: "ok" },
{	another: "ok" },
{	two: "ok" },
{	two: "ok" },
{	two: "ok" }
]`)
	target := filepath.Join(t.TempDir(), "test.cue")
	err := os.WriteFile(target, source, 0700)
	if err != nil {
		t.Fatal(err)
	}
	p, err := AppendToStructList(source, cuecontext.New().BuildExpr(
		ast.NewStruct(
			&ast.Field{
				Label: ast.NewString("Name"),
				Value: ast.NewString("Someone"),
			},
			&ast.Field{
				Label: ast.NewString("Email"),
				Value: ast.NewString("someEmail@somehost.net"),
			},
		),
	))
	if err != nil {
		t.Fatal(err)
	}
	// time.Sleep(time.Second * 2)
	if err = Commit(target, filepath.Dir(target), p); err != nil {
		t.Fatal(err)
	}

	result, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(result))
	// t.Fatal("check")
}

func ensureInversible(
	source []byte,
	patch Patch,
) func(*testing.T) {
	return func(t *testing.T) {
		intermediate, err := patch.ApplyToCueSource(source)
		if err != nil {
			t.Log("original:", string(source))
			t.Log("intermediate:", string(intermediate))
			t.Fatal("unable to apply patch:", err)
		}
		reversed, err := patch.Invert().ApplyToCueSource(intermediate)
		if err != nil {
			t.Fatal("unable to apply reverse:", err)
		}
		if string(source) != string(reversed) {
			t.Log("original:", string(source))
			t.Log("intermediate:", string(intermediate))
			t.Log("reversed:", string(reversed))
			t.Fatal("reversed patch does not match original")
		}
	}
}
