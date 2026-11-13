package htmx

import (
	"context"
	"errors"
	"strconv"

	"github.com/dkotik/cuebook"
)

type moveRequest struct {
	File        string `schema:"file"`
	From        string `schema:"from"`
	To          string `schema:"to"`
	Destination string `schema:"destination"`
}

func (*moveRequest) Validate(context.Context) error { return nil }

type moveUUIDRequest struct {
	File        string `schema:"file"`
	FromUUID    string `schema:"from_uuid"`
	To          string `schema:"to"`
	Destination string `schema:"destination"`
}

func (*moveUUIDRequest) Validate(context.Context) error { return nil }

type moveResponse struct {
	pageData
}

func (a *handler) move(ctx context.Context, request *moveRequest) (moveResponse, error) {
	if a.files == nil {
		return moveResponse{}, ErrSourceReadOnly
	}
	fileNames, raw, document, err := a.loadEntryDocument(request.File)
	if err != nil {
		return moveResponse{}, err
	}
	from, err := strconv.Atoi(request.From)
	if err != nil {
		return moveResponse{}, ErrInvalidEntryPosition
	}
	return a.moveEntry(ctx, request.File, fileNames, raw, document, from, request.To, request.Destination)
}

func (a *handler) moveUUID(ctx context.Context, request *moveUUIDRequest) (moveResponse, error) {
	if a.files == nil {
		return moveResponse{}, ErrSourceReadOnly
	}
	fileNames, raw, document, err := a.loadEntryDocument(request.File)
	if err != nil {
		return moveResponse{}, err
	}
	from, err := entryIndexByUUID(document, request.FromUUID)
	if errors.Is(err, cuebook.ErrBookEntryNotFound) {
		return moveResponse{}, &cuebook.EntryNotFoundError{Path: request.File}
	}
	if err != nil {
		return moveResponse{}, err
	}
	return a.moveEntry(ctx, request.File, fileNames, raw, document, from, request.To, request.Destination)
}

func (a *handler) moveEntry(ctx context.Context, fileName string, fileNames []string, raw []byte, document cuebook.Book, from int, toPosition, destination string) (moveResponse, error) {
	length, err := document.Len()
	if err != nil {
		return moveResponse{}, CauseError{Code: ErrUnableToReadEntryCount, Cause: err}
	}
	if from < 0 || from >= length {
		return moveResponse{}, &cuebook.EntryNotFoundError{Path: fileName}
	}
	if destination != "" {
		data, err := a.transferEntry(ctx, fileName, destination, raw, document, from, fileNames)
		return moveResponse{pageData: data}, err
	}

	to, err := strconv.Atoi(toPosition)
	if err != nil {
		return moveResponse{}, ErrInvalidEntryPosition
	}
	if to < 0 || to >= length {
		return moveResponse{}, &cuebook.EntryNotFoundError{Path: fileName}
	}
	if from == to {
		data, err := a.renderEditedFile(fileName)
		return moveResponse{pageData: data}, err
	}

	change, err := document.MoveEntry(from, to)
	if err != nil {
		return moveResponse{}, CauseError{Code: ErrUnableToReorderEntries, Cause: err}
	}
	candidate, err := change.ApplyToCueSource(raw)
	if err != nil {
		return moveResponse{}, err
	}
	if _, err := cuebook.New(candidate); err != nil {
		return moveResponse{}, CauseError{Code: ErrReorderedDocumentConstraintViolation, Cause: err}
	}
	if err := a.commitFile(ctx, fileName, change); err != nil {
		return moveResponse{}, err
	}
	data, err := a.renderEditedFile(fileName)
	return moveResponse{pageData: data}, err
}

func (a *handler) transferEntry(ctx context.Context, sourceName, destinationName string, sourceRaw []byte, sourceDocument cuebook.Book, from int, fileNames []string) (pageData, error) {
	if sourceName == destinationName {
		return a.renderEditedFile(sourceName)
	}

	destinationRaw, _, err := a.readDocument(destinationName, fileNames)
	if err != nil {
		return pageData{}, err
	}
	sourceEntry, err := sourceDocument.GetEntryRef(from)
	if err != nil {
		return pageData{}, &cuebook.EntryNotFoundError{Path: sourceName}
	}
	transfer, err := cuebook.PrepareEntryTransfer(destinationRaw, sourceDocument, sourceEntry)
	if err != nil {
		return pageData{}, CauseError{Code: ErrUnableToPrepareEntryTransfer, Cause: err}
	}
	appendChange := transfer.DestinationPatch
	destinationCandidate, err := appendChange.ApplyToCueSource(destinationRaw)
	if err != nil {
		return pageData{}, CauseError{Code: ErrDestinationChangedBeforeTransfer, Cause: err}
	}
	if _, err := cuebook.New(destinationCandidate); err != nil {
		return pageData{}, CauseError{Code: ErrDestinationConstraintViolation, Cause: err}
	}

	deleteChange := transfer.SourcePatch
	sourceCandidate, err := deleteChange.ApplyToCueSource(sourceRaw)
	if err != nil {
		return pageData{}, CauseError{Code: ErrSourceChangedBeforeTransfer, Cause: err}
	}
	if _, err := cuebook.New(sourceCandidate); err != nil {
		return pageData{}, CauseError{Code: ErrSourceConstraintViolation, Cause: err}
	}

	if err := a.commitFile(ctx, destinationName, cuebook.Validated(appendChange)); err != nil {
		return pageData{}, err
	}
	if err := a.commitFile(ctx, sourceName, cuebook.Validated(deleteChange)); err != nil {
		// Roll back even if the request was canceled; ctx no longer matters once
		// the destination has been changed.
		return pageData{}, errors.Join(
			err,
			a.commitFile(context.WithoutCancel(ctx), destinationName, cuebook.Validated(appendChange.Invert())),
		)
	}
	return a.renderEditedFile(destinationName)
}
