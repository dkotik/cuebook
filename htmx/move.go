package htmx

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/patch"
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

func (a *handler) move(_ context.Context, request *moveRequest) (moveResponse, error) {
	if a.committer == nil {
		return moveResponse{}, errors.New("This source is read-only.")
	}

	fileName := request.File
	fileNames, err := a.fileNames()
	if err != nil {
		return moveResponse{}, errors.New("Unable to list CUE files.")
	}
	raw, document, status, message := a.readDocument(fileName, fileNames)
	if status != http.StatusOK {
		return moveResponse{}, documentReadError(fileName, status, message)
	}

	from, err := strconv.Atoi(request.From)
	if err != nil {
		return moveResponse{}, errors.New("The entry position is invalid.")
	}
	length, err := document.Len()
	if err != nil {
		return moveResponse{}, errors.New("Unable to read the entry count.")
	}
	if from < 0 || from >= length {
		return moveResponse{}, &cuebook.ItemNotFoundError{Path: fileName}
	}
	if request.Destination != "" {
		data, err := a.transferEntry(fileName, request.Destination, raw, document, from, fileNames)
		return moveResponse{pageData: data}, err
	}

	to, err := strconv.Atoi(request.To)
	if err != nil {
		return moveResponse{}, errors.New("The entry position is invalid.")
	}
	if to < 0 || to >= length {
		return moveResponse{}, &cuebook.ItemNotFoundError{Path: fileName}
	}
	if from == to {
		data, err := a.renderEditedFile(fileName)
		return moveResponse{pageData: data}, err
	}

	change, err := entryMovePatch(raw, from, to)
	if err != nil {
		return moveResponse{}, errors.New("The entries could not be reordered.")
	}
	candidate, err := change.ApplyToCueSource(raw)
	if err != nil {
		if errors.Is(err, patch.ErrByteRangeNotFound) {
			return moveResponse{}, errors.New("The document changed before the entries could be reordered. Reload and try again.")
		}
		return moveResponse{}, errors.New("The entries could not be reordered.")
	}
	if _, err := cuebook.New(candidate); err != nil {
		return moveResponse{}, errors.New("The reordered document does not satisfy the CUE constraints.")
	}
	if err := a.commitFile(fileName, change); err != nil {
		if errors.Is(err, patch.ErrByteRangeNotFound) {
			return moveResponse{}, errors.New("The document changed before the entries could be reordered. Reload and try again.")
		}
		return moveResponse{}, errors.New("The entries could not be saved.")
	}
	data, err := a.renderEditedFile(fileName)
	return moveResponse{pageData: data}, err
}

func (a *handler) transferEntry(sourceName, destinationName string, sourceRaw []byte, sourceDocument cuebook.Book, from int, fileNames []string) (pageData, error) {
	if sourceName == destinationName {
		return a.renderEditedFile(sourceName)
	}

	destinationRaw, _, status, message := a.readDocument(destinationName, fileNames)
	if status != http.StatusOK {
		return pageData{}, documentReadError(destinationName, status, message)
	}
	sourceEntryValue, err := sourceDocument.GetValue(from)
	if err != nil {
		return pageData{}, &cuebook.ItemNotFoundError{Path: sourceName}
	}
	destinationEntryValue, _, err := entryValueWithoutConstraints(sourceEntryValue)
	if err != nil {
		return pageData{}, errors.New("The source entry could not be converted for transfer.")
	}

	appendChange, err := patch.AppendToStructList(destinationRaw, destinationEntryValue)
	if err != nil {
		return pageData{}, errors.New("The entry could not be added to the destination file.")
	}
	destinationCandidate, err := appendChange.ApplyToCueSource(destinationRaw)
	if err != nil {
		return pageData{}, errors.New("The destination file changed before the entry could be added. Reload and try again.")
	}
	if _, err := cuebook.New(destinationCandidate); err != nil {
		return pageData{}, errors.New("The entry does not satisfy the destination file's CUE constraints.")
	}

	deleteChange, err := patch.DeleteFromStructList(sourceRaw, sourceEntryValue)
	if err != nil {
		return pageData{}, errors.New("The entry could not be removed from the source file.")
	}
	sourceCandidate, err := deleteChange.ApplyToCueSource(sourceRaw)
	if err != nil {
		return pageData{}, errors.New("The source file changed before the entry could be removed. Reload and try again.")
	}
	if _, err := cuebook.New(sourceCandidate); err != nil {
		return pageData{}, errors.New("Removing the entry does not satisfy the source file's CUE constraints.")
	}

	if err := a.commitFile(destinationName, patch.Validated(appendChange)); err != nil {
		if errors.Is(err, patch.ErrByteRangeNotFound) {
			return pageData{}, errors.New("The destination file changed before the entry could be added. Reload and try again.")
		}
		return pageData{}, errors.New("The entry could not be added to the destination file.")
	}
	if err := a.commitFile(sourceName, patch.Validated(deleteChange)); err != nil {
		rollbackErr := a.commitFile(destinationName, patch.Validated(appendChange.Invert()))
		if rollbackErr != nil {
			return pageData{}, errors.New("The entry was added to the destination, but the source could not be updated and the destination change could not be rolled back. It may now exist in both files.")
		}
		if errors.Is(err, patch.ErrByteRangeNotFound) {
			return pageData{}, errors.New("The source file changed before the entry could be removed. The destination change was rolled back; reload and try again.")
		}
		return pageData{}, errors.New("The transfer could not be completed; the destination change was rolled back.")
	}
	return a.renderEditedFile(destinationName)
}

func entryValueWithoutConstraints(value cue.Value) (cue.Value, string, error) {
	encoded, err := value.MarshalJSON()
	if err != nil {
		return cue.Value{}, "", err
	}

	intermediate := string(encoded)
	decoded := cuecontext.New().CompileString(intermediate)
	if err := decoded.Err(); err != nil {
		return cue.Value{}, intermediate, err
	}
	return decoded, intermediate, nil
}

func entryMovePatch(source []byte, from, to int) (patch.Patch, error) {
	if from == to {
		return nil, nil
	}

	current := append([]byte(nil), source...)
	steps := make([]patch.Patch, 0, max(from-to, to-from))
	direction := 1
	if from > to {
		direction = -1
	}

	for index := from; index != to; index += direction {
		compiled := cuecontext.New().CompileBytes(current)
		earlier := compiled.LookupPath(cue.MakePath(cue.Index(index)))
		later := compiled.LookupPath(cue.MakePath(cue.Index(index + direction)))
		change, err := patch.SwapEntries(current, earlier, later)
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
	steps []patch.Patch
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

func (p swapSequencePatch) Invert() patch.Patch {
	inverse := make([]patch.Patch, 0, len(p.steps))
	for i := len(p.steps) - 1; i >= 0; i-- {
		inverse = append(inverse, p.steps[i].Invert())
	}
	return swapSequencePatch{steps: inverse}
}
