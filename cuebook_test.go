package cuebook

import "testing"

func TestValueIterator(t *testing.T) {
	book, err := New([]byte(`[
		{ "key": "value" },
		{ "key2": "value2" }
	]`))
	if err != nil {
		t.Fatal(err)
	}

	var entries []string
	for entry := range book.EachValue() {
		b, err := entry.MarshalJSON()
		if err != nil {
			t.Fatal(err)
		}
		entries = append(entries, string(b))
	}

	if len(entries) != 2 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0] != `{"key":"value"}` {
		t.Fatalf(`expected entry to be {"key":"value"}, got %s`, entries[0])
	}
	if entries[1] != `{"key2":"value2"}` {
		t.Fatalf(`expected entry to be {"key2":"value2"}, got %s`, entries[0])
	}
}
