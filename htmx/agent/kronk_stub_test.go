//go:build !kronk

package agent

import (
	"context"
	"strings"
	"testing"
)

func TestKronkRequiresBuildTag(t *testing.T) {
	_, err := newKronkGenerator(context.Background(), defaultModelSource, t.TempDir(), 128)
	if err == nil || !strings.Contains(err.Error(), "-tags kronk") {
		t.Fatalf("newKronkGenerator error = %v, want build-tag guidance", err)
	}
}
