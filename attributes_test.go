package cuebook

import (
	"bytes"
	"encoding/json"
	"reflect"
	"regexp"
	"testing"
)

func TestPatchesDoNotCopySchemaAttributes(t *testing.T) {
	t.Parallel()
	const (
		sourceSchema = `#entry: {
	@cuebook(entry)
	Name:     string @cuebook(title)
	Notes?:   string @cuebook(multiline)
	Enabled?: bool   @cuebook(label="Is enabled")
	Address?: {
		@cuebook(group)
		City: string @cuebook(title)
	}
}
`
		destinationSchema = `#entry: {
	Name:     string @cuebook(label="Full name")
	Notes?:   string @other(notes)
	Enabled?: bool
	Address?: {City: string @other(city)}
}
`
		// The second entry carries hand-written attributes. In-place rewrites
		// keep them; appended copies are pure data.
		entries = `[...#entry] & [
	{Name: "first", Notes: "kept"},
	{@note(entry), Name: "second", Notes: "old", Enabled: true @legacy(flag), Address: {City: "Paris" @legacy(city)}},
]
`
	)
	writtenAttributes := []string{"@note(entry)", "@legacy(flag)", "@legacy(city)"}
	attributePattern := regexp.MustCompile(`@\w+\([^)]*\)`)

	tests := []struct {
		name string
		// destination is the schema of the file that receives the patch. The
		// entry always comes from a file using sourceSchema.
		destination string
		patch       func(destination []byte, entry Entry) (Patch, error)
		// index and want describe the written entry in the patched file.
		index int
		want  string
		// attributes are the only attributes expected in the written content.
		attributes []string
	}{
		{
			name:        "append entry",
			destination: sourceSchema,
			patch: func(destination []byte, entry Entry) (Patch, error) {
				book, err := New(destination)
				if err != nil {
					return nil, err
				}
				return book.AppendToStructList(entry.Value)
			},
			index: 2,
			want:  `{"Name": "second", "Notes": "old", "Enabled": true, "Address": {"City": "Paris"}}`,
		},
		{
			name:        "append entry to file with similar schema",
			destination: destinationSchema,
			patch: func(destination []byte, entry Entry) (Patch, error) {
				book, err := New(destination)
				if err != nil {
					return nil, err
				}
				return book.AppendToStructList(entry.Value)
			},
			index: 2,
			want:  `{"Name": "second", "Notes": "old", "Enabled": true, "Address": {"City": "Paris"}}`,
		},
		{
			name:        "update field",
			destination: sourceSchema,
			patch: func(_ []byte, entry Entry) (Patch, error) {
				return entry.UpdateFieldValue("Name", "renamed")
			},
			index:      1,
			want:       `{"Name": "renamed", "Notes": "old", "Enabled": true, "Address": {"City": "Paris"}}`,
			attributes: writtenAttributes,
		},
		{
			name:        "update multiline field",
			destination: sourceSchema,
			patch: func(_ []byte, entry Entry) (Patch, error) {
				return entry.UpdateFieldValue("Notes", "line one\nline two")
			},
			index:      1,
			want:       `{"Name": "second", "Notes": "line one\nline two", "Enabled": true, "Address": {"City": "Paris"}}`,
			attributes: writtenAttributes,
		},
		{
			name:        "remove field",
			destination: sourceSchema,
			patch: func(destination []byte, entry Entry) (Patch, error) {
				return entry.RemoveFieldValue("Notes")
			},
			index:      1,
			want:       `{"Name": "second", "Enabled": true, "Address": {"City": "Paris"}}`,
			attributes: writtenAttributes,
		},
		{
			name:        "merge field values",
			destination: sourceSchema,
			patch: func(_ []byte, entry Entry) (Patch, error) {
				return entry.MergeFieldValues(map[string]string{
					"Name":  "merged",
					"Notes": "line one\nline two",
				})
			},
			index:      1,
			want:       `{"Name": "merged", "Notes": "line one\nline two", "Enabled": true, "Address": {"City": "Paris"}}`,
			attributes: writtenAttributes,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			book, err := New([]byte(sourceSchema + entries))
			if err != nil {
				t.Fatal(err)
			}
			entry, err := book.GetEntry(1)
			if err != nil {
				t.Fatal(err)
			}
			destination := []byte(tt.destination + entries)
			change, err := tt.patch(destination, entry)
			if err != nil {
				t.Fatal(err)
			}
			updated, err := change.ApplyToCueSource(destination)
			if err != nil {
				t.Fatal(err)
			}

			written := change.Difference().Content
			var attributes []string
			for _, attribute := range attributePattern.FindAll(written, -1) {
				attributes = append(attributes, string(attribute))
			}
			if !reflect.DeepEqual(attributes, tt.attributes) {
				t.Fatalf("written attributes = %q, want %q:\n%s", attributes, tt.attributes, written)
			}
			if !bytes.HasPrefix(updated, []byte(tt.destination)) {
				t.Fatalf("schema attributes were not preserved:\n%s", updated)
			}
			patched, err := New(updated)
			if err != nil {
				t.Fatalf("invalid result: %v\n%s", err, updated)
			}
			writtenEntry, err := patched.GetValue(tt.index)
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := writtenEntry.MarshalJSON()
			if err != nil {
				t.Fatal(err)
			}
			var got, want any
			if err := json.Unmarshal(encoded, &got); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(tt.want), &want); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("written entry = %s, want %s\n%s", encoded, tt.want, updated)
			}
			t.Run("reverse", ensureInversible(destination, change))
		})
	}
}
