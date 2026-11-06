package cuebook

import (
	"fmt"
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
// to move one entry between CUE lists.
type EntryTransfer struct {
	SourcePatch      Patch
	DestinationPatch Patch
}

// PrepareEntryTransfer creates patches to append entry to the destination list
// and remove it from the source list. The caller is responsible for validating
// and committing both documents together.
func PrepareEntryTransfer(source, destination []byte, entry cue.Value) (EntryTransfer, error) {
	var transfer EntryTransfer

	unconstrained, _, err := EntryValueWithoutConstraints(entry)
	if err != nil {
		return transfer, fmt.Errorf("unable to convert source entry for transfer: %w", err)
	}
	transfer.DestinationPatch, err = AppendToStructList(destination, unconstrained)
	if err != nil {
		return transfer, fmt.Errorf("unable to append entry to destination: %w", err)
	}
	transfer.SourcePatch, err = DeleteFromStructList(source, entry)
	if err != nil {
		return transfer, fmt.Errorf("unable to remove entry from source: %w", err)
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
