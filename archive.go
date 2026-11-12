package cuebook

import (
	"bytes"
	"time"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
)

const (
	// ArchiveDirectory is the directory used for archived CUE lists.
	ArchiveDirectory = ".archive"
	// EmptyArchiveSource is the initial content of a new archive list.
	EmptyArchiveSource = "[]\n"
)

// ArchiveFileName returns the archive file for the given date.
func ArchiveFileName(date time.Time) string {
	return ArchiveDirectory + "/" + date.Format("2006-01-02") + ".cue"
}

// EntryTransfer contains the reversible source and destination changes needed
// to move one entry between CUE lists. Retain and reuse these exact patches when
// retrying. Preparing a new transfer while a previous one may have completed
// only one side is ambiguous, so duplicate destination entries are rejected.
type EntryTransfer struct {
	SourcePatch      Patch
	DestinationPatch Patch
}

// PrepareEntryTransfer creates patches to append entry to the destination list
// and remove it from the source list. The caller is responsible for validating
// and committing both documents together.
func PrepareEntryTransfer(destination []byte, entry Entry) (EntryTransfer, error) {
	var transfer EntryTransfer

	unconstrained, _, err := EntryValueWithoutConstraints(entry.Value)
	if err != nil {
		return transfer, err
	}
	destinationBook, err := New(destination)
	if err != nil {
		return transfer, err
	}
	transferredJSON, err := unconstrained.MarshalJSON()
	if err != nil {
		return transfer, err
	}
	destinationValues, err := destinationBook.Value.List()
	if err != nil {
		return transfer, err
	}
	for destinationValues.Next() {
		existingJSON, err := destinationValues.Value().MarshalJSON()
		if err != nil {
			return transfer, err
		}
		if bytes.Equal(existingJSON, transferredJSON) {
			return transfer, ErrPatchTargetAmbiguous
		}
	}
	transfer.DestinationPatch, err = destinationBook.AppendToStructList(unconstrained)
	if err != nil {
		return transfer, err
	}
	transfer.SourcePatch, err = entry.DeleteFromStructList()
	if err != nil {
		return transfer, err
	}
	return transfer, nil
}

// EntryValueWithoutConstraints returns a CUE value with the same concrete JSON
// data but without references to the source document's definitions.
func EntryValueWithoutConstraints(value cue.Value) (cue.Value, string, error) {
	encoded, err := value.MarshalJSON()
	if err != nil {
		return cue.Value{}, "", err
	}

	intermediate := string(encoded)
	decoded := cuecontext.New().CompileString(intermediate)
	if err := decoded.Err(); err != nil {
		return cue.Value{}, intermediate, err
	}
	return decoded, intermediate, nil
}
