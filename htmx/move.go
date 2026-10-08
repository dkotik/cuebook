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

func (a *handler) move(ctx context.Context, request *moveRequest) (moveResponse, error) {
	if a.committer == nil {
		return moveResponseFrom(a.editFailure(ctx, "", "This source is read-only.", http.StatusForbidden))
	}

	fileName := request.File
	fileNames, err := a.fileNames()
	if err != nil {
		return moveResponseFrom(a.editFailure(ctx, "", "Unable to list CUE files.", http.StatusInternalServerError))
	}
	raw, document, status, message := a.readDocument(fileName, fileNames)
	if status != http.StatusOK {
		return moveResponseFrom(a.editFailure(ctx, fileName, message, status))
	}

	from, err := strconv.Atoi(request.From)
	length, _ := document.Len()
	if err != nil || from < 0 || from >= length {
		return moveResponseFrom(a.editFailure(ctx, fileName, "The entry position is invalid.", http.StatusBadRequest))
	}
	if request.Destination != "" {
		return moveResponseFrom(a.transferEntry(ctx, fileName, request.Destination, raw, document, from, fileNames))
	}

	to, err := strconv.Atoi(request.To)
	if err != nil || to < 0 || to >= length {
		return moveResponseFrom(a.editFailure(ctx, fileName, "The entry position is invalid.", http.StatusBadRequest))
	}
	if from == to {
		return moveResponseFrom(a.finishEdit(fileName))
	}

	change, err := entryMovePatch(raw, from, to)
	if err != nil {
		return moveResponseFrom(a.editFailure(ctx, fileName, "The entries could not be reordered.", http.StatusUnprocessableEntity))
	}
	candidate, err := change.ApplyToCueSource(raw)
	if err != nil {
		status = http.StatusConflict
		notice := "The document changed before the entries could be reordered. Reload and try again."
		if !errors.Is(err, patch.ErrByteRangeNotFound) {
			status = http.StatusUnprocessableEntity
			notice = "The entries could not be reordered."
		}
		return moveResponseFrom(a.editFailure(ctx, fileName, notice, status))
	}
	if _, err := cuebook.New(candidate); err != nil {
		return moveResponseFrom(a.editFailure(ctx, fileName, "The reordered document does not satisfy the CUE constraints.", http.StatusUnprocessableEntity))
	}
	if err := a.commitFile(fileName, change); err != nil {
		status := http.StatusInternalServerError
		notice := "The entries could not be saved."
		if errors.Is(err, patch.ErrByteRangeNotFound) {
			status = http.StatusConflict
			notice = "The document changed before the entries could be reordered. Reload and try again."
		}
		return moveResponseFrom(a.editFailure(ctx, fileName, notice, status))
	}
	return moveResponseFrom(a.finishEdit(fileName))
}

func (a *handler) transferEntry(ctx context.Context, sourceName, destinationName string, sourceRaw []byte, sourceDocument cuebook.Book, from int, fileNames []string) (pageData, int) {
	if sourceName == destinationName {
		return a.finishEdit(sourceName)
	}

	destinationRaw, _, status, message := a.readDocument(destinationName, fileNames)
	if status != http.StatusOK {
		return a.editFailure(ctx, destinationName, message, status)
	}
	sourceEntryValue, err := sourceDocument.GetValue(from)
	if err != nil {
		return a.editFailure(ctx, sourceName, "The source entry could not be found.", http.StatusNotFound)
	}
	destinationEntryValue, _, err := entryValueWithoutConstraints(sourceEntryValue)
	if err != nil {
		return a.editFailure(ctx, sourceName, "The source entry could not be converted for transfer.", http.StatusUnprocessableEntity)
	}

	appendChange, err := patch.AppendToStructList(destinationRaw, destinationEntryValue)
	if err != nil {
		return a.editFailure(ctx, destinationName, "The entry could not be added to the destination file.", http.StatusUnprocessableEntity)
	}
	destinationCandidate, err := appendChange.ApplyToCueSource(destinationRaw)
	if err != nil {
		return a.editFailure(ctx, destinationName, "The destination file changed before the entry could be added. Reload and try again.", http.StatusConflict)
	}
	if _, err := cuebook.New(destinationCandidate); err != nil {
		return a.editFailure(ctx, destinationName, "The entry does not satisfy the destination file's CUE constraints.", http.StatusUnprocessableEntity)
	}

	deleteChange, err := patch.DeleteFromStructList(sourceRaw, sourceEntryValue)
	if err != nil {
		return a.editFailure(ctx, sourceName, "The entry could not be removed from the source file.", http.StatusUnprocessableEntity)
	}
	sourceCandidate, err := deleteChange.ApplyToCueSource(sourceRaw)
	if err != nil {
		return a.editFailure(ctx, sourceName, "The source file changed before the entry could be removed. Reload and try again.", http.StatusConflict)
	}
	if _, err := cuebook.New(sourceCandidate); err != nil {
		return a.editFailure(ctx, sourceName, "Removing the entry does not satisfy the source file's CUE constraints.", http.StatusUnprocessableEntity)
	}

	if err := a.commitFile(destinationName, patch.Validated(appendChange)); err != nil {
		status := http.StatusInternalServerError
		notice := "The entry could not be added to the destination file."
		if errors.Is(err, patch.ErrByteRangeNotFound) {
			status = http.StatusConflict
			notice = "The destination file changed before the entry could be added. Reload and try again."
		}
		return a.editFailure(ctx, destinationName, notice, status)
	}
	if err := a.commitFile(sourceName, patch.Validated(deleteChange)); err != nil {
		rollbackErr := a.commitFile(destinationName, patch.Validated(appendChange.Invert()))
		if rollbackErr != nil {
			return a.editFailure(ctx, destinationName, "The entry was added to the destination, but the source could not be updated and the destination change could not be rolled back. It may now exist in both files.", http.StatusInternalServerError)
		}
		status := http.StatusInternalServerError
		notice := "The transfer could not be completed; the destination change was rolled back."
		if errors.Is(err, patch.ErrByteRangeNotFound) {
			status = http.StatusConflict
			notice = "The source file changed before the entry could be removed. The destination change was rolled back; reload and try again."
		}
		return a.editFailure(ctx, sourceName, notice, status)
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
