package identifier

import (
	"net/url"
	"strings"
	"testing"
	"uuid"
)

func TestUUIDGeneration(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
	}{
		{name: "without prefix"},
		{name: "with prefix", prefix: "prefix"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			id, err := GenerateUUID("", url.Values{"prefix": {tt.prefix}})
			if err != nil {
				t.Fatal("unable to generate UUID:", err)
			}
			if !strings.HasPrefix(id, tt.prefix) {
				t.Fatalf("generated ID %q does not start with prefix %q", id, tt.prefix)
			}
			generated := strings.TrimPrefix(id, tt.prefix)
			if _, err := uuid.Parse(generated); err != nil {
				t.Fatalf("generated suffix %q is not a UUID: %v", generated, err)
			}
		})
	}
}
