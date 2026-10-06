package search

import (
	"testing"

	"github.com/dkotik/cuebook"
)

func TestBleveSearch(t *testing.T) {
	t.Parallel()

	book, err := cuebook.New([]byte(`[
		{ "key": "valueGreat" },
		{ "key2": "value2" }
	]`))
	if err != nil {
		t.Fatal(err)
	}
	index := NewBleveIndex()
	const filePath = "nested/testFile.cue"

	var ranges []cuebook.ByteRange
	for entry, err := range book.EachEntry() {
		if err != nil {
			t.Fatal("entry loading failed:", err)
		}
		byteRange, err := cuebook.NewByteRange(entry.Value)
		if err != nil {
			t.Fatal("entry range lookup failed:", err)
		}
		if err = index.Include(filePath, entry); err != nil {
			t.Fatal("entry indexing failed:", err)
		}
		ranges = append(ranges, byteRange)
	}
	if len(ranges) != 2 {
		t.Fatalf("indexed entry count = %d, want 2", len(ranges))
	}

	tests := []struct {
		name           string
		query          string
		wantCount      int
		wantRange      cuebook.ByteRange
		wantField      string
		wantFieldValue string
	}{
		{
			name:           "result contains first entry and source range",
			query:          "valueGreat",
			wantCount:      1,
			wantRange:      ranges[0],
			wantField:      "key",
			wantFieldValue: "valueGreat",
		},
		{
			name:           "result contains second entry and source range",
			query:          "value2",
			wantCount:      1,
			wantRange:      ranges[1],
			wantField:      "key2",
			wantFieldValue: "value2",
		},
		{
			name:      "query with no matches returns no results",
			query:     "not-present",
			wantCount: 0,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			results, err := index.Query(test.query)
			if err != nil {
				t.Fatal("search query failed:", err)
			}
			if len(results) != test.wantCount {
				t.Fatalf("search result count = %d, want %d", len(results), test.wantCount)
			}
			if test.wantCount == 0 {
				return
			}

			result := results[0]
			if result.Path != filePath {
				t.Errorf("result path = %q, want %q", result.Path, filePath)
			}
			if result.ByteRange != test.wantRange {
				t.Errorf("result byte range = %+v, want %+v", result.ByteRange, test.wantRange)
			}
			field, ok := result.Entry.GetFieldByName(test.wantField)
			if !ok {
				t.Fatalf("result entry does not contain field %q", test.wantField)
			}
			if got := field.String(); got != test.wantFieldValue {
				t.Errorf("result field value = %q, want %q", got, test.wantFieldValue)
			}
		})
	}
}
