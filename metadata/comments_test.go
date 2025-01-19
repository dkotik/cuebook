package metadata

import (
	"bytes"
	"testing"
)

func TestExtractLeadingComments(t *testing.T) {
	cases := [...]struct {
		Source    []byte
		Comments  []byte
		BytesRead int
	}{
		{
			Source:    nil,
			Comments:  nil,
			BytesRead: 0,
		},
		{
			Source:    []byte("/"),
			Comments:  nil,
			BytesRead: 0,
		},
		{
			Source:    []byte("//"),
			Comments:  nil,
			BytesRead: 2,
		},
		{
			Source:    []byte("// first"),
			Comments:  []byte("first\n"),
			BytesRead: 8,
		},
		{
			Source:    []byte("// first\n"),
			Comments:  []byte("first\n"),
			BytesRead: 9,
		},
		{
			Source:    []byte("// first\n// second"),
			Comments:  []byte("first\nsecond\n"),
			BytesRead: 18,
		},
		{
			Source:    []byte("// first\n// second\nend"),
			Comments:  []byte("first\nsecond\n"),
			BytesRead: 19,
		},
		{
			Source:    []byte("\n\n// first\n// second\nend"),
			Comments:  []byte("\nfirst\nsecond\n"),
			BytesRead: 21,
		},
		{
			Source:    []byte("\n\n// first\n\n//\n\n\n\n// second\nend"),
			Comments:  []byte("\nfirst\n\nsecond\n"),
			BytesRead: 28,
		},
	}

	for i, tc := range cases {
		comments, read := ReadLeadingComments(tc.Source)
		if !bytes.Equal(tc.Comments, comments) {
			t.Log("given:", string(comments))
			t.Log("expected:", string(tc.Comments))
			t.Fatalf("case %d failed: extracted comment does not match the expected", i+1)
		} else if tc.BytesRead != read {
			t.Log("given:", read)
			t.Log("expected:", tc.BytesRead)
			t.Fatalf("case %d failed: comment bytes read does not match", i+1)
		}
	}
}
