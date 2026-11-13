package htmx

import (
	"context"
	"errors"
	"io/fs"
	"strconv"
	"strings"
	"time"

	"github.com/dkotik/cuebook"
)

const (
	archiveDirectory = cuebook.ArchiveDirectory + "/"
	archiveFolder    = cuebook.ArchiveDirectory
)

type archiveRequest struct {
	File  string `schema:"file"`
	Entry string `schema:"entry"`
}

func (*archiveRequest) Validate(context.Context) error { return nil }

type archiveUUIDRequest struct {
	File string `schema:"file"`
	UUID string `schema:"uuid"`
}

func (*archiveUUIDRequest) Validate(context.Context) error { return nil }

type archiveResponse struct {
	pageData
}

func (a *handler) archive(ctx context.Context, request *archiveRequest) (archiveResponse, error) {
	if a.files == nil {
		return archiveResponse{}, ErrSourceReadOnly
	}
	if strings.HasPrefix(request.File, archiveDirectory) {
		return archiveResponse{}, ErrEntryAlreadyArchived
	}
	_, raw, document, err := a.loadEntryDocument(request.File)
	if err != nil {
		return archiveResponse{}, err
	}
	from, err := strconv.Atoi(request.Entry)
	if err != nil {
		return archiveResponse{}, ErrInvalidEntryPosition
	}
	return a.archiveEntry(ctx, request.File, raw, document, from)
}

func (a *handler) archiveUUID(ctx context.Context, request *archiveUUIDRequest) (archiveResponse, error) {
	if a.files == nil {
		return archiveResponse{}, ErrSourceReadOnly
	}
	if strings.HasPrefix(request.File, archiveDirectory) {
		return archiveResponse{}, ErrEntryAlreadyArchived
	}
	_, raw, document, err := a.loadEntryDocument(request.File)
	if err != nil {
		return archiveResponse{}, err
	}
	from, err := entryIndexByUUID(document, request.UUID)
	if errors.Is(err, cuebook.ErrBookEntryNotFound) {
		return archiveResponse{}, &cuebook.EntryNotFoundError{Path: request.File}
	}
	if err != nil {
		return archiveResponse{}, err
	}
	return a.archiveEntry(ctx, request.File, raw, document, from)
}

func (a *handler) archiveEntry(ctx context.Context, fileName string, raw []byte, document cuebook.Book, from int) (archiveResponse, error) {
	length, lengthErr := document.Len()
	if lengthErr != nil {
		return archiveResponse{}, CauseError{Code: ErrUnableToReadEntryCount, Cause: lengthErr}
	}
	if from < 0 || from >= length {
		return archiveResponse{}, &cuebook.EntryNotFoundError{Path: fileName}
	}

	archiveName := cuebook.ArchiveFileName(time.Now())
	if err := a.createFileIfNotExists(ctx, archiveName, []byte(cuebook.EmptyArchiveSource)); err != nil {
		return archiveResponse{}, CauseError{Code: ErrUnableToCreateArchiveFile, Cause: err}
	}

	fileNames, err := a.fileNames()
	if err != nil {
		return archiveResponse{}, ErrUnableToListCueFiles
	}
	if !containsFile(fileNames, archiveName) {
		return archiveResponse{}, &cuebook.FileSystemPathError{PathError: &fs.PathError{
			Path: archiveName, Err: cuebook.ErrFileSystemFileNotFound,
		}}
	}

	data, err := a.transferEntry(ctx, fileName, archiveName, raw, document, from, fileNames)
	return archiveResponse{pageData: data}, err
}
