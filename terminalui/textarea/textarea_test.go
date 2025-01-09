package textarea

import (
	"testing"
)

func TestInitialization(t *testing.T) {
	ta := New("standard", "Label", "value", false)
	if ta.(Textarea).status == nil {
		t.Error("status model is nil")
	}
}
