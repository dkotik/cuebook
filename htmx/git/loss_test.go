package git

import (
	"reflect"
	"testing"
)

func TestCountLostLines(t *testing.T) {
	tests := []struct {
		name         string
		addedLines   []string
		removedLines []string
		want         []string
	}{
		{
			name:         "re-added lines are not lost",
			addedLines:   []string{"kept", "new"},
			removedLines: []string{"kept", "old"},
			want:         []string{"old"},
		},
		{
			name:         "unmatched repeated removals are preserved",
			removedLines: []string{"old", "old"},
			want:         []string{"old", "old"},
		},
		{
			name:         "empty lines are compared as content",
			addedLines:   []string{""},
			removedLines: []string{""},
			want:         nil,
		},
		{
			name: "no removals",
			want: nil,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := countLostLines(test.addedLines, test.removedLines)
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("countLostLines() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestParseDiffLineChanges(t *testing.T) {
	tests := []struct {
		name        string
		output      string
		wantAdded   []string
		wantRemoved []string
	}{
		{
			name: "ignores diff metadata and reads hunk content",
			output: "diff --git a/file.txt b/file.txt\n" +
				"--- a/file.txt\n+++ b/file.txt\n" +
				"@@ -1,2 +1,2 @@\n-old\n+new\n" +
				"\\ No newline at end of file\n",
			wantAdded:   []string{"new"},
			wantRemoved: []string{"old"},
		},
		{
			name: "preserves prefixes that resemble diff headers within hunks",
			output: "diff --git a/file.txt b/file.txt\n" +
				"--- a/file.txt\n+++ b/file.txt\n" +
				"@@ -1,2 +1,2 @@\n---- content\n++++ content\n",
			wantAdded:   []string{"+++ content"},
			wantRemoved: []string{"--- content"},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			gotAdded, gotRemoved := parseDiffLineChanges([]byte(test.output))
			if !reflect.DeepEqual(gotAdded, test.wantAdded) {
				t.Errorf("added lines = %#v, want %#v", gotAdded, test.wantAdded)
			}
			if !reflect.DeepEqual(gotRemoved, test.wantRemoved) {
				t.Errorf("removed lines = %#v, want %#v", gotRemoved, test.wantRemoved)
			}
		})
	}
}
