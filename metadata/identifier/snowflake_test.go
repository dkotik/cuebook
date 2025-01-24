package identifier

import (
	"net/url"
	"strings"
	"testing"
)

func TestSnowFlakeIDGeneration(t *testing.T) {
	id, err := GenerateSnowFlakeID("prefix", url.Values{})
	if err != nil {
		t.Fatal("unable to generate new snowflake ID:", err)
	}
	if !strings.HasPrefix(id, "prefix") {
		t.Fatal("input prefix was not included in ID generation", id)
	}
}
