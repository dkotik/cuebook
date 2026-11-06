package htmx

import (
	"context"
	"errors"
	"io/fs"
	"net/http"
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
		return archiveResponse{}, errors.New("This source is read-only.")
	}

	fileName := request.File
	if strings.HasPrefix(fileName, archiveDirectory) {
		return archiveResponse{}, errors.New("Entries in the archive cannot be archived again.")
	}

	fileNames, err := a.fileNames()
	if err != nil {
		return archiveResponse{}, errors.New("Unable to list CUE files.")
	}
	raw, document, status, message := a.readDocument(fileName, fileNames)
	if status != http.StatusOK {
		return archiveResponse{}, documentReadError(fileName, status, message)
	}

	from, err := strconv.Atoi(request.Entry)
	if err != nil {
		return archiveResponse{}, errors.New("The entry position is invalid.")
	}
	length, lengthErr := document.Len()
	if lengthErr != nil {
		return archiveResponse{}, errors.New("Unable to read the entry count.")
	}
	if from < 0 || from >= length {
		return archiveResponse{}, &cuebook.EntryNotFoundError{Path: fileName}
	}

	archiveName := cuebook.ArchiveFileName(time.Now())
	if err := a.createFileIfNotExists(ctx, archiveName, []byte(cuebook.EmptyArchiveSource)); err != nil {
		return archiveResponse{}, errors.New("Unable to create the archive file.")
	}

	fileNames, err = a.fileNames()
	if err != nil {
		return archiveResponse{}, errors.New("Unable to list CUE files.")
	}
	if !containsFile(fileNames, archiveName) {
		return archiveResponse{}, &cuebook.FileSystemPathError{PathError: &fs.PathError{
			Path: archiveName, Err: cuebook.ErrFileSystemFileNotFound,
		}}
	}

	data, err := a.transferEntry(ctx, fileName, archiveName, raw, document, from, fileNames)
	return archiveResponse{pageData: data}, err
}
