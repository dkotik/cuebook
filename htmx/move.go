package htmx

import (
	"context"
	"errors"
	"strconv"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/dkotik/cuebook"
)

type moveRequest struct {
	File        string `schema:"file"`
	From        string `schema:"from"`
	To          string `schema:"to"`
	Destination string `schema:"destination"`
}

func (*moveRequest) Validate(context.Context) error { return nil }

type moveResponse struct {
	pageData
}

func (a *handler) move(ctx context.Context, request *moveRequest) (moveResponse, error) {
	if a.files == nil {
		return moveResponse{}, ErrSourceReadOnly
	}

	fileName := request.File
	fileNames, err := a.fileNames()
	if err != nil {
		return moveResponse{}, ErrUnableToListCueFiles
	}
	raw, document, err := a.readDocument(fileName, fileNames)
	if err != nil {
		return moveResponse{}, err
	}

	from, err := strconv.Atoi(request.From)
	if err != nil {
		return moveResponse{}, ErrInvalidEntryPosition
	}
	length, err := document.Len()
	if err != nil {
		return moveResponse{}, CauseError{Code: ErrUnableToReadEntryCount, Cause: err}
	}
	if from < 0 || from >= length {
		return moveResponse{}, &cuebook.EntryNotFoundError{Path: fileName}
	}
	if request.Destination != "" {
		data, err := a.transferEntry(ctx, fileName, request.Destination, raw, document, from, fileNames)
		return moveResponse{pageData: data}, err
	}

	to, err := strconv.Atoi(request.To)
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

	change, err := entryMovePatch(raw, from, to)
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
	sourceEntryValue, err := sourceDocument.GetValue(from)
	if err != nil {
		return pageData{}, &cuebook.EntryNotFoundError{Path: sourceName}
	}
	transfer, err := cuebook.PrepareEntryTransfer(sourceRaw, destinationRaw, sourceEntryValue)
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

func entryMovePatch(source []byte, from, to int) (cuebook.Patch, error) {
	if from == to {
		return nil, nil
	}

	current := append([]byte(nil), source...)
	steps := make([]cuebook.Patch, 0, max(from-to, to-from))
	direction := 1
	if from > to {
		direction = -1
	}

	for index := from; index != to; index += direction {
		compiled := cuecontext.New().CompileBytes(current)
		earlier := compiled.LookupPath(cue.MakePath(cue.Index(index)))
		later := compiled.LookupPath(cue.MakePath(cue.Index(index + direction)))
		change, err := cuebook.SwapEntries(current, earlier, later)
		if err != nil {
			return nil, err
		}
		current, err = change.ApplyToCueSource(current)
		if err != nil {
			return nil, err
		}
		steps = append(steps, change)
	}
	if len(steps) == 1 {
		return steps[0], nil
	}
	return swapSequencePatch{steps: steps}, nil
}

type swapSequencePatch struct {
	steps []cuebook.Patch
}

func (p swapSequencePatch) ApplyToCueSource(source []byte) ([]byte, error) {
	var err error
	for _, step := range p.steps {
		source, err = step.ApplyToCueSource(source)
		if err != nil {
			return nil, err
		}
	}
	return source, nil
}

func (p swapSequencePatch) Difference() cuebook.ByteAnchor {
	if len(p.steps) == 0 {
		return cuebook.ByteAnchor{}
	}
	return p.steps[len(p.steps)-1].Difference()
}

func (p swapSequencePatch) Invert() cuebook.Patch {
	inverse := make([]cuebook.Patch, 0, len(p.steps))
	for i := len(p.steps) - 1; i >= 0; i-- {
		inverse = append(inverse, p.steps[i].Invert())
	}
	return swapSequencePatch{steps: inverse}
}
