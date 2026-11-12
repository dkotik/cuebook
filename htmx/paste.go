package htmx

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strconv"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/dkotik/cuebook"
)

type pasteRequest struct {
	File           string `schema:"file"`
	Source         string `schema:"source"`
	CutFile        string `schema:"cut_file"`
	CutEntry       string `schema:"cut_entry"`
	CutFingerprint string `schema:"cut_fingerprint"`
}

func (*pasteRequest) Validate(context.Context) error { return nil }

func (a *handler) paste(ctx context.Context, request *pasteRequest) (listResponse, error) {
	if a.files == nil {
		return listResponse{}, ErrSourceReadOnly
	}
	fileNames, err := a.fileNames()
	if err != nil {
		return listResponse{}, ErrUnableToListCueFiles
	}
	raw, _, err := a.readDocument(request.File, fileNames)
	if err != nil {
		return listResponse{}, err
	}
	if strings.TrimSpace(request.Source) == "" {
		return listResponse{}, ErrPastedSourceEmpty
	}
	entry := cuecontext.New().CompileString(request.Source)
	if err := entry.Err(); err != nil {
		return listResponse{}, CauseError{Code: ErrUnableToParsePastedCUEStruct, Cause: err}
	}
	if entry.IncompleteKind() != cue.StructKind {
		return listResponse{}, cuebook.ErrEntryValueNotStruct
	}
	if request.CutFile != "" || request.CutEntry != "" || request.CutFingerprint != "" {
		if request.CutFile == "" || request.CutEntry == "" || request.CutFingerprint == "" {
			return listResponse{}, ErrIncompleteCutInformation
		}
		// A clipboard changed outside Cuebook must not move the previously cut entry.
		if entrySourceFingerprint(request.Source) == request.CutFingerprint {
			return a.pasteCut(ctx, request, fileNames)
		}
	}
	if err := a.appendEntry(ctx, request.File, raw, entry); err != nil {
		return listResponse{}, err
	}
	data, err := a.renderEditedFile(request.File)
	return listResponse{pageData: data}, err
}

func (a *handler) pasteCut(ctx context.Context, request *pasteRequest, fileNames []string) (listResponse, error) {
	index, err := strconv.Atoi(request.CutEntry)
	if err != nil || index < 0 {
		return listResponse{}, ErrInvalidCutEntryPosition
	}
	raw, document, err := a.readDocument(request.CutFile, fileNames)
	if err != nil {
		return listResponse{}, err
	}
	value, err := document.GetValueByIndex(index)
	if err != nil {
		return listResponse{}, cuebook.ErrBookEntryIndexOutOfRange
	}
	source, err := entryCUESource(value)
	if err != nil || entrySourceFingerprint(source) != request.CutFingerprint {
		return listResponse{}, ErrCutEntryChangedOrMoved
	}
	data, err := a.transferEntry(ctx, request.CutFile, request.File, raw, document, index, fileNames)
	return listResponse{pageData: data}, err
}

func entrySourceFingerprint(source string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(strings.ReplaceAll(source, "\r\n", "\n"))))
}
