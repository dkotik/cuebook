package htmx

import (
	"context"
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

type archiveResponse struct {
	pageData
}

func (a *handler) archive(ctx context.Context, request *archiveRequest) (archiveResponse, error) {
	if a.files == nil {
		return archiveResponse{}, ErrSourceReadOnly
	}

	fileName := request.File
	if strings.HasPrefix(fileName, archiveDirectory) {
		return archiveResponse{}, ErrEntryAlreadyArchived
	}

	fileNames, err := a.fileNames()
	if err != nil {
		return archiveResponse{}, ErrUnableToListCueFiles
	}
	raw, document, err := a.readDocument(fileName, fileNames)
	if err != nil {
		return archiveResponse{}, err
	}

	from, err := strconv.Atoi(request.Entry)
	if err != nil {
		return archiveResponse{}, ErrInvalidEntryPosition
	}
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

	fileNames, err = a.fileNames()
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
