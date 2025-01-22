package cuebook

import "testing"

func TestBleveSearch(t *testing.T) {
	book, err := New([]byte(`[
		{ "key": "valueGreat" },
		{ "key2": "value2" }
	]`))
	if err != nil {
		t.Fatal(err)
	}
	index := NewBleveIndex()

	i := 0
	for entry, err := range book.EachEntry() {
		if err != nil {
			t.Fatal("entry loading failed:", err)
		}
		if err = index.Include(IndexKey{
			Index:    i,
			FilePath: "testFile",
		}, entry); err != nil {
			t.Fatal("entry indexing failed:", err)
		}
		i++
	}

	searchResults, err := index.Query("valueGreat")
	if err != nil {
		t.Fatal("search query failed:", err)
	}
	t.Log(searchResults)

	if len(searchResults) != 1 {
		t.Fatalf("unexpected search result count: %d vs expected 1", len(searchResults))
	}
	if searchResults[0].Fields[0].String() != "valueGreat" {
		t.Fatal("search result mismatch")
	}

	// t.Fatal("implement")
}
