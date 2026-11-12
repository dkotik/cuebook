package internal

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"time"

	"github.com/dkotik/cuebook"
)

func archiveEntry(ctx context.Context, files cuebook.FileSystem, sourceName string, entry cuebook.Entry, date time.Time) (result cuebook.Result, err error) {
	archiveName := cuebook.ArchiveFileName(date)
	archiveSource, err := files.ReadFile(ctx, archiveName)
	createArchive := errors.Is(err, fs.ErrNotExist)
	if err != nil && !createArchive {
		return result, fmt.Errorf("unable to read archive file %q: %w", archiveName, err)
	}
	if createArchive {
		archiveSource = []byte(cuebook.EmptyArchiveSource)
	}

	transfer, err := cuebook.PrepareEntryTransfer(archiveSource, entry)
	if err != nil {
		return result, err
	}

	if createArchive {
		if err = files.WriteFile(ctx, archiveName, []byte(cuebook.EmptyArchiveSource)); err != nil {
			return result, fmt.Errorf("unable to create archive file %q: %w", archiveName, err)
		}
	}
	if _, err = cuebook.Commit(ctx, files, archiveName, transfer.DestinationPatch); err != nil {
		return result, fmt.Errorf("unable to append entry to archive %q: %w", archiveName, err)
	}

	// Keep the archived copy if updating the source fails; duplication is safer
	// than rolling back the only durable copy of the entry.
	result, err = cuebook.Commit(ctx, files, sourceName, transfer.SourcePatch)
	if err != nil {
		return result, fmt.Errorf("unable to remove entry from source %q: %w", sourceName, err)
	}
	return result, nil
}
