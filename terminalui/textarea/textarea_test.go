package textarea

import (
	"testing"
)

func TestInitialization(t *testing.T) {
	type testType string

	ta, err := New(WithLabel("Label"))
	if err != nil {
		t.Fatal(err)
	}
	t.Log(ta)
}
