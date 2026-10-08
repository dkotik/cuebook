package htmx

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"
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

func (a *handler) archive(ctx context.Context, request *archiveRequest) (archiveResponse, error) {
	if a.committer == nil {
		return archiveResponseFrom(a.editFailure(ctx, "", "This source is read-only.", http.StatusForbidden))
	}

	creator, ok := a.committer.(FileCreator)
	if !ok {
		return archiveResponseFrom(a.editFailure(ctx, "", "This source cannot create archive files.", http.StatusNotImplemented))
	}

	fileName := request.File
	if strings.HasPrefix(fileName, archiveDirectory) {
		return archiveResponseFrom(a.editFailure(ctx, fileName, "Entries in the archive cannot be deleted.", http.StatusForbidden))
	}

	fileNames, err := a.fileNames()
	if err != nil {
		return archiveResponseFrom(a.editFailure(ctx, "", "Unable to list CUE files.", http.StatusInternalServerError))
	}
	raw, document, status, message := a.readDocument(fileName, fileNames)
	if status != http.StatusOK {
		return archiveResponseFrom(a.editFailure(ctx, fileName, message, status))
	}

	from, err := strconv.Atoi(request.Entry)
	length, lengthErr := document.Len()
	if err != nil || lengthErr != nil || from < 0 || from >= length {
		return archiveResponseFrom(a.editFailure(ctx, fileName, "The entry position is invalid.", http.StatusBadRequest))
	}

	archiveName := archiveDirectory + time.Now().Format("2006-01-02") + ".cue"
	if err := a.applyFileChange(archiveName, func() error {
		return creator.CreateFileIfNotExists(archiveName, []byte("[]\n"))
	}); err != nil {
		return archiveResponseFrom(a.editFailure(ctx, fileName, "Unable to create the archive file.", http.StatusInternalServerError))
	}

	fileNames, err = a.fileNames()
	if err != nil {
		return archiveResponseFrom(a.editFailure(ctx, fileName, "Unable to list CUE files.", http.StatusInternalServerError))
	}
	if !containsFile(fileNames, archiveName) {
		return archiveResponseFrom(a.editFailure(ctx, fileName, "The archive file is not available in this source.", http.StatusInternalServerError))
	}

	return archiveResponseFrom(a.transferEntry(ctx, fileName, archiveName, raw, document, from, fileNames))
}
