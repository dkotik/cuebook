package htmx

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
)

type pasteRequest struct {
	File           string `schema:"file"`
	Source         string `schema:"source"`
	CutFile        string `schema:"cut_file"`
	CutEntry       string `schema:"cut_entry"`
	CutFingerprint string `schema:"cut_fingerprint"`
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
	if request.CutFile != "" || request.CutEntry != "" || request.CutFingerprint != "" {
		if request.CutFile == "" || request.CutEntry == "" || request.CutFingerprint == "" {
			return listResponse{}, errors.New("The cut information is incomplete. Copy or cut the entry again.")
		}
		// A clipboard changed outside Cuebook must not move the previously cut entry.
		if entrySourceFingerprint(request.Source) == request.CutFingerprint {
			return a.pasteCut(request, fileNames)
		}
	}
	data, err := a.appendEntry(request.File, raw, entry)
	return listResponse{pageData: data}, err
}

func (a *handler) pasteCut(request *pasteRequest, fileNames []string) (listResponse, error) {
	index, err := strconv.Atoi(request.CutEntry)
	if err != nil || index < 0 {
		return listResponse{}, errors.New("The cut entry position is invalid. Cut the entry again.")
	}
	raw, document, status, message := a.readDocument(request.CutFile, fileNames)
	if status != http.StatusOK {
		return listResponse{}, documentReadError(request.CutFile, status, message)
	}
	value, err := document.GetValue(index)
	if err != nil {
		return listResponse{}, errors.New("The cut entry no longer exists. Copy or cut the entry again.")
	}
	source, err := entryCUESource(value)
	if err != nil || entrySourceFingerprint(source) != request.CutFingerprint {
		return listResponse{}, errors.New("The cut entry changed or moved. Copy or cut the entry again.")
	}
	data, err := a.transferEntry(request.CutFile, request.File, raw, document, index, fileNames)
	return listResponse{pageData: data}, err
}

func entrySourceFingerprint(source string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(strings.ReplaceAll(source, "\r\n", "\n"))))
}
