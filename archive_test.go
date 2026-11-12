package cuebook

import (
	"bytes"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestArchiveFileName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		date time.Time
		want string
	}{
		{
			name: "date in archive path",
			date: time.Date(2026, time.November, 6, 23, 59, 0, 0, time.UTC),
			want: ".archive/2026-11-06.cue",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := ArchiveFileName(test.date); got != test.want {
				t.Errorf("archive file name = %q, want %q", got, test.want)
			}
		})
	}
}

func TestPrepareEntryTransfer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		source      string
		destination string
		wantDest    []string
	}{
		{
			name: "removes source and appends unconstrained value",
			source: `#entry: {Name: string @cuebook(title), Optional?: bool}
[...#entry] & [{Name: "Move me", Optional: true}, {Name: "Keep me"}]
`,
			destination: "[]\n",
			wantDest:    []string{"Move me"},
		},
		{
			name: "preserves existing destination entries",
			source: `#entry: {Name: string @cuebook(title)}
[...#entry] & [{Name: "Move me"}, {Name: "Keep me"}]
`,
			destination: `#entry: {Name: string @cuebook(title)}
[...#entry] & [{Name: "Already archived"}]
`,
			wantDest: []string{"Already archived", "Move me"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			source := []byte(test.source)
			sourceBook, err := New(source)
			if err != nil {
				t.Fatal(err)
			}
			entry, err := sourceBook.GetEntry(0)
			if err != nil {
				t.Fatal(err)
			}

			transfer, err := PrepareEntryTransfer([]byte(test.destination), entry)
			if err != nil {
				t.Fatal(err)
			}
			updatedSource, err := transfer.SourcePatch.ApplyToCueSource(source)
			if err != nil {
				t.Fatal(err)
			}
			updatedDestination, err := transfer.DestinationPatch.ApplyToCueSource([]byte(test.destination))
			if err != nil {
				t.Fatal(err)
			}
			retriedSource, err := transfer.SourcePatch.ApplyToCueSource(updatedSource)
			if err != nil || !bytes.Equal(retriedSource, updatedSource) {
				t.Fatalf("source transfer retry = %q, err = %v; want unchanged source", retriedSource, err)
			}
			retriedDestination, err := transfer.DestinationPatch.ApplyToCueSource(updatedDestination)
			if err != nil || !bytes.Equal(retriedDestination, updatedDestination) {
				t.Fatalf("destination transfer retry = %q, err = %v; want unchanged destination", retriedDestination, err)
			}
			if _, err := New(updatedSource); err != nil {
				t.Fatalf("updated source is invalid: %v", err)
			}
			if _, err := New(updatedDestination); err != nil {
				t.Fatalf("updated destination is invalid: %v", err)
			}

			if got := titlesInList(t, updatedSource); !slices.Equal(got, []string{"Keep me"}) {
				t.Errorf("source titles = %q, want [Keep me]", got)
			}
			if got := titlesInList(t, updatedDestination); !slices.Equal(got, test.wantDest) {
				t.Errorf("destination titles = %q, want %q", got, test.wantDest)
			}
		})
	}
}

func TestPrepareEntryTransferRejectsAmbiguousDuplicate(t *testing.T) {
	source := []byte(`[{Name: "Move me"}]`)
	book, err := New(source)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := book.GetEntry(0)
	if err != nil {
		t.Fatal(err)
	}

	_, err = PrepareEntryTransfer([]byte(`[{Name: "Move me"}]`), entry)
	if !errors.Is(err, ErrPatchTargetAmbiguous) {
		t.Fatalf("PrepareEntryTransfer() error = %v, want %v", err, ErrPatchTargetAmbiguous)
	}
}

func TestEntryValueWithoutConstraints(t *testing.T) {
	t.Parallel()

	book, err := New([]byte(`#entry: {Name: string @cuebook(title)}
[...#entry] & [{Name: "Ada"}]
`))
	if err != nil {
		t.Fatal(err)
	}
	entry, err := book.GetValue(0)
	if err != nil {
		t.Fatal(err)
	}

	decoded, intermediate, err := EntryValueWithoutConstraints(entry)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(intermediate, "#entry") {
		t.Fatalf("intermediate value retained a source definition reference: %s", intermediate)
	}
	originalJSON, err := entry.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	decodedJSON, err := decoded.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decodedJSON, originalJSON) {
		t.Fatalf("decoded value = %s, want %s", decodedJSON, originalJSON)
	}
}

func titlesInList(t *testing.T, source []byte) []string {
	t.Helper()
	book, err := New(source)
	if err != nil {
		t.Fatal(err)
	}
	var titles []string
	for entry, err := range book.EachEntry() {
		if err != nil {
			t.Fatal(err)
		}
		titles = append(titles, entry.GetTitle())
	}
	return titles
}
