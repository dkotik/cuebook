package htmx

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dkotik/cuebook"
)

const archiveDirectory = ".archive/"

type archiveRequest struct {
	File  string `schema:"file"`
	Entry string `schema:"entry"`
}

func (*archiveRequest) Validate(context.Context) error { return nil }

type archiveResponse struct {
	pageData
}

func (a *handler) archive(_ context.Context, request *archiveRequest) (archiveResponse, error) {
	if a.committer == nil {
		return archiveResponse{}, errors.New("This source is read-only.")
	}

	creator, ok := a.committer.(FileCreator)
	if !ok {
		return archiveResponse{}, errors.New("This source cannot create archive files.")
	}

	fileName := request.File
	if strings.HasPrefix(fileName, archiveDirectory) {
		return archiveResponse{}, errors.New("Entries in the archive cannot be deleted.")
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
		return archiveResponse{}, &cuebook.ItemNotFoundError{Path: fileName}
	}

	archiveName := archiveDirectory + time.Now().Format("2006-01-02") + ".cue"
	if err := a.applyFileChange(archiveName, func() error {
		return creator.CreateFileIfNotExists(archiveName, []byte("[]\n"))
	}); err != nil {
		return archiveResponse{}, errors.New("Unable to create the archive file.")
	}

	fileNames, err = a.fileNames()
	if err != nil {
		return archiveResponse{}, errors.New("Unable to list CUE files.")
	}
	if !containsFile(fileNames, archiveName) {
		return archiveResponse{}, &cuebook.FileNotFoundError{Path: archiveName}
	}

	data, err := a.transferEntry(fileName, archiveName, raw, document, from, fileNames)
	return archiveResponse{pageData: data}, err
}
