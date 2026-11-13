package internal

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/terminalui/internal/list"
)

func TestEntryListHidesConfiguredUUIDFromCardDescription(t *testing.T) {
	const source = `#Entry: {
ID: string @cuebook(uuid)
OtherID: string @cuebook(uuid)
Name: string @cuebook(title)
}
[...#Entry] & [{ID: "primary-id", OtherID: "secondary-id", Name: "Ada"}]`
	book, err := cuebook.New([]byte(source))
	if err != nil {
		t.Fatal(err)
	}

	message := LoadEntries(cuebook.Result{Document: book, Source: []byte(source)}, 0)()
	cards, ok := message.(entryListCards)
	if !ok {
		t.Fatalf("LoadEntries() returned %T, want entryListCards", message)
	}
	entryCard, ok := cards.Cards[1].(Entry)
	if !ok {
		t.Fatalf("entry card = %T, want Entry", cards.Cards[1])
	}
	model, _ := entryCard.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	content := model.View().Content
	if strings.Contains(content, "primary-id") {
		t.Errorf("entry list card displays the configured UUID: %s", content)
	}
	if !strings.Contains(content, "secondary-id") {
		t.Errorf("entry list card should retain later UUID-marked fields: %s", content)
	}
}

func TestEntryListMixedIdentityReorderUsesUUID(t *testing.T) {
	tests := []struct {
		name            string
		source          string
		duplicateSource string
		currentIndex    int
		desiredIndex    int
		wantFirstUUID   string
	}{
		{
			name: "identified entry moves down past anonymous entry",
			source: `#entry: {ID?: string @cuebook(uuid), Name: string}
[...#entry] & [{ID: "target", Name: "Identified"}, {Name: "Anonymous"}]`,
			duplicateSource: `#entry: {ID?: string @cuebook(uuid), Name: string}
[...#entry] & [{ID: "target", Name: "Identified"}, {Name: "Anonymous"}, {ID: "target", Name: "Duplicate"}]`,
			currentIndex:  1,
			desiredIndex:  2,
			wantFirstUUID: "",
		},
		{
			name: "identified entry moves up past anonymous entry",
			source: `#entry: {ID?: string @cuebook(uuid), Name: string}
[...#entry] & [{Name: "Anonymous"}, {ID: "target", Name: "Identified"}]`,
			duplicateSource: `#entry: {ID?: string @cuebook(uuid), Name: string}
[...#entry] & [{Name: "Anonymous"}, {ID: "target", Name: "Identified"}, {ID: "target", Name: "Duplicate"}]`,
			currentIndex:  2,
			desiredIndex:  1,
			wantFirstUUID: "target",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			book, err := cuebook.New([]byte(tt.source))
			if err != nil {
				t.Fatal(err)
			}
			model, ok := NewEntryList("").(EntryList)
			if !ok {
				t.Fatalf("NewEntryList() returned %T, want EntryList", model)
			}
			model.book = cuebook.Result{Document: book, Source: []byte(tt.source)}

			_, cmd := model.Update(list.SwapOrderEvent{
				CurrentIndex: tt.currentIndex,
				DesiredIndex: tt.desiredIndex,
			})
			if cmd == nil {
				t.Fatal("reorder did not produce a command")
			}
			result := cmd()
			patch, ok := result.(cuebook.Patch)
			if !ok {
				t.Fatalf("reorder returned %T, want cuebook.Patch", result)
			}
			updated, err := patch.ApplyToCueSource([]byte(tt.source))
			if err != nil {
				t.Fatal(err)
			}
			updatedBook, err := cuebook.New(updated)
			if err != nil {
				t.Fatal(err)
			}
			first, err := updatedBook.GetEntryByIndex(0)
			if err != nil {
				t.Fatal(err)
			}
			if first.UUID != tt.wantFirstUUID {
				t.Errorf("first entry UUID = %q, want %q", first.UUID, tt.wantFirstUUID)
			}

			if _, err := patch.ApplyToCueSource([]byte(tt.duplicateSource)); !errors.Is(err, cuebook.ErrPatchTargetAmbiguous) {
				t.Errorf("applying reorder to duplicate UUIDs error = %v, want %v", err, cuebook.ErrPatchTargetAmbiguous)
			}
		})
	}
}
