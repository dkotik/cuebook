package cuebook

import "testing"

func TestNewEntryPopulatesFieldDescriptionsFromPrecedingComments(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		description string
	}{
		{
			name: "single comment",
			source: `[
				{
					// The contact's display name.
					Name: "Ada",
				}
			]`,
			description: "The contact's display name.",
		},
		{
			name: "multiple comment lines",
			source: `[
				{
					// Primary contact email.
					// Use an address that can receive messages.
					Email: "ada@example.test",
				}
			]`,
			description: "Primary contact email.\nUse an address that can receive messages.",
		},
		{
			name:   "no comment",
			source: `[{Name: "Ada"}]`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			book, err := New([]byte(test.source))
			if err != nil {
				t.Fatal(err)
			}
			for entry, err := range book.EachEntry() {
				if err != nil {
					t.Fatal(err)
				}
				field, ok := entry.GetFieldByName("Name")
				if !ok {
					field, ok = entry.GetFieldByName("Email")
				}
				if !ok {
					t.Fatal("field not found")
				}
				if field.Description != test.description {
					t.Errorf("field description = %q, want %q", field.Description, test.description)
				}
				return
			}
			t.Fatal("entry not found")
		})
	}
}
