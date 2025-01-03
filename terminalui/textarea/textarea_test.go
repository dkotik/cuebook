package textarea

import (
	"testing"
)

func TestInitialization(t *testing.T) {
	ta, _ := Textarea{
		Label:    "Enter field",
		Required: true,
	}.Init()
	if ta.(Textarea).status == nil {
		t.Error("status model is nil")
	}
}
