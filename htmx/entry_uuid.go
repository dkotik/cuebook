package htmx

import (
	"github.com/dkotik/cuebook"
)

func entryIndexByUUID(book cuebook.Book, uuid string) (int, error) {
	if uuid == "" {
		return -1, cuebook.ErrBookEntryNotFound
	}

	found := -1
	index := 0
	for entry, err := range book.EachEntry() {
		if err != nil {
			return -1, err
		}
		if entry.UUID == uuid {
			if found >= 0 {
				return -1, cuebook.ErrPatchTargetAmbiguous
			}
			found = index
		}
		index++
	}
	if found < 0 {
		return -1, cuebook.ErrBookEntryNotFound
	}
	return found, nil
}
