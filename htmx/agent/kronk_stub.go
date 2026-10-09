//go:build !kronk

package agent

import (
	"context"
	"errors"
)

func newKronkGenerator(_ context.Context, _, _ string, _ int) (Generator, error) {
	return nil, errors.New("agent: Kronk support is not enabled in this build; rebuild with -tags kronk and install the required native libffi runtime")
}
