package cuebook

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
	result, err := Commit(target, filepath.Dir(target), p)
	if err != nil {
		t.Fatal(err)
	}
	if result.LastChange == nil {
		t.Fatal("Commit() result has no last change")
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(result.Source) {
		t.Fatalf("committed source = %q, result source = %q", got, result.Source)
	}
	if count, err := result.Document.Len(); err != nil || count != 7 {
		t.Fatalf("committed document length = %d, err = %v; want 7", count, err)
	}
}

func TestResultBottomChangeIndex(t *testing.T) {
	tests := []struct {
		name          string
		currentSource string
		olderSource   string
		want          int
	}{
		{
			name:          "unchanged document",
			currentSource: `[{Name: "one"}, {Name: "two"}]`,
			olderSource:   `[{Name: "one"}, {Name: "two"}]`,
			want:          -1,
		},
		{
			name:          "first entry changed",
			currentSource: `[{Name: "updated"}, {Name: "two"}]`,
			olderSource:   `[{Name: "one"}, {Name: "two"}]`,
			want:          0,
		},
		{
			name:          "later entry changed",
			currentSource: `[{Name: "one"}, {Name: "updated"}]`,
			olderSource:   `[{Name: "one"}, {Name: "two"}]`,
			want:          1,
		},
		{
			name:          "new entry appended",
			currentSource: `[{Name: "one"}, {Name: "two"}]`,
			olderSource:   `[{Name: "one"}]`,
			want:          1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			currentSource := []byte(test.currentSource)
			current, err := New(currentSource)
			if err != nil {
				t.Fatal(err)
			}
			olderSource := []byte(test.olderSource)
			older, err := New(olderSource)
			if err != nil {
				t.Fatal(err)
			}
			got := (Result{Document: current, Source: currentSource}).BottomChangeIndex(
				Result{Document: older, Source: olderSource},
			)
			if got != test.want {
				t.Fatalf("BottomChangeIndex() = %d, want %d", got, test.want)
			}
		})
	}
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
