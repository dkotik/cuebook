package patch_test

import (
	"bytes"
	"testing"

	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/patch"
)

func TestCompatibilityPackageForwardsToCore(t *testing.T) {
	var _ cuebook.Patch = patch.Patch(nil)
	if patch.ErrByteRangeNotFound != cuebook.ErrByteRangeNotFound {
		t.Fatal("legacy byte-range error is not the core sentinel")
	}

	source := []byte("abc")
	change, err := patch.UpdateRange(source, cuebook.ByteRange{Head: 1, Tail: 2}, []byte("B"))
	if err != nil {
		t.Fatal(err)
	}
	updated, err := change.ApplyToCueSource(source)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(updated, []byte("aBc")) {
		t.Fatalf("legacy API result = %q, want %q", updated, "aBc")
	}
}
