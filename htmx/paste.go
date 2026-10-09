package htmx

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
)

type pasteRequest struct {
	File   string `schema:"file"`
	Source string `schema:"source"`
}

func (*pasteRequest) Validate(context.Context) error { return nil }

func (a *handler) paste(_ context.Context, request *pasteRequest) (listResponse, error) {
	if a.committer == nil {
		return listResponse{}, errors.New("This source is read-only.")
	}
	fileNames, err := a.fileNames()
	if err != nil {
		return listResponse{}, errors.New("Unable to list CUE files.")
	}
	raw, _, status, message := a.readDocument(request.File, fileNames)
	if status != http.StatusOK {
		return listResponse{}, documentReadError(request.File, status, message)
	}
	if strings.TrimSpace(request.Source) == "" {
		return listResponse{}, errors.New("The pasted source is empty.")
	}
	entry := cuecontext.New().CompileString(request.Source)
	if err := entry.Err(); err != nil {
		return listResponse{}, errors.New("Unable to parse the pasted CUE struct: " + err.Error())
	}
	if entry.IncompleteKind() != cue.StructKind {
		return listResponse{}, errors.New("The pasted source must be a CUE struct.")
	}
	data, err := a.appendEntry(request.File, raw, entry)
	return listResponse{pageData: data}, err
}
