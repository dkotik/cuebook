package identifier

import (
	"net/url"
	"strings"
	"testing"
)

func TestUUIDGeneration(t *testing.T) {
	id, err := GenerateUUID("", url.Values{
		"prefix": []string{"prefix"},
	})
	if err != nil {
		t.Fatal("unable to generate new snowflake ID:", err)
	}
	if !strings.HasPrefix(id, "prefix") {
		t.Fatal("input prefix was not included in ID generation", id)
	}
}
