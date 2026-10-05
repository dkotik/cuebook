package htmx

import (
	"errors"
	"net/http"
	"strconv"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/patch"
)

func (a *handler) move(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(r) {
		a.renderPage(w, r, pageData{ReadOnly: a.committer == nil, Error: "Cross-origin edits are not allowed."}, http.StatusForbidden)
		return
	}
	if a.committer == nil {
		a.editFailure(w, r, "", "This source is read-only.", http.StatusForbidden)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := r.ParseForm(); err != nil {
		a.editFailure(w, r, "", "The move request is invalid.", http.StatusBadRequest)
		return
	}

	fileName := r.PostForm.Get("file")
	fileNames, err := a.fileNames()
	if err != nil {
		a.editFailure(w, r, "", "Unable to list CUE files.", http.StatusInternalServerError)
		return
	}
	raw, document, status, message := a.readDocument(fileName, fileNames)
	if status != http.StatusOK {
		a.editFailure(w, r, fileName, message, status)
		return
	}

	from, err := strconv.Atoi(r.PostForm.Get("from"))
	length, _ := document.Len()
	if err != nil || from < 0 || from >= length {
		a.editFailure(w, r, fileName, "The entry position is invalid.", http.StatusBadRequest)
		return
	}
	if destination := r.PostForm.Get("destination"); destination != "" {
		a.transferEntry(w, r, fileName, destination, raw, document, from, fileNames)
		return
	}

	to, err := strconv.Atoi(r.PostForm.Get("to"))
	if err != nil || to < 0 || to >= length {
		a.editFailure(w, r, fileName, "The entry position is invalid.", http.StatusBadRequest)
		return
	}
	if from == to {
		a.finishEdit(w, r, fileName)
		return
	}

	change, err := entryMovePatch(raw, from, to)
	if err != nil {
		a.editFailure(w, r, fileName, "The entries could not be reordered.", http.StatusUnprocessableEntity)
		return
	}
	candidate, err := change.ApplyToCueSource(raw)
	if err != nil {
		status = http.StatusConflict
		notice := "The document changed before the entries could be reordered. Reload and try again."
		if !errors.Is(err, patch.ErrByteRangeNotFound) {
			status = http.StatusUnprocessableEntity
			notice = "The entries could not be reordered."
		}
		a.editFailure(w, r, fileName, notice, status)
		return
	}
	if _, err := cuebook.New(candidate); err != nil {
		a.editFailure(w, r, fileName, "The reordered document does not satisfy the CUE constraints.", http.StatusUnprocessableEntity)
		return
	}
	if err := a.committer.Commit(fileName, change); err != nil {
		status := http.StatusInternalServerError
		notice := "The entries could not be saved."
		if errors.Is(err, patch.ErrByteRangeNotFound) {
			status = http.StatusConflict
			notice = "The document changed before the entries could be reordered. Reload and try again."
		}
		a.editFailure(w, r, fileName, notice, status)
		return
	}
	a.finishEdit(w, r, fileName)
}

func (a *handler) transferEntry(w http.ResponseWriter, r *http.Request, sourceName, destinationName string, sourceRaw []byte, sourceDocument cuebook.Document, from int, fileNames []string) {
	if sourceName == destinationName {
		a.finishEdit(w, r, sourceName)
		return
	}

	destinationRaw, _, status, message := a.readDocument(destinationName, fileNames)
	if status != http.StatusOK {
		a.editFailure(w, r, destinationName, message, status)
		return
	}
	sourceEntryValue, err := sourceDocument.GetValue(from)
	if err != nil {
		a.editFailure(w, r, sourceName, "The source entry could not be found.", http.StatusNotFound)
		return
	}
	destinationEntryValue, _, err := entryValueWithoutConstraints(sourceEntryValue)
	if err != nil {
		a.editFailure(w, r, sourceName, "The source entry could not be converted for transfer.", http.StatusUnprocessableEntity)
		return
	}

	appendChange, err := patch.AppendToStructList(destinationRaw, destinationEntryValue)
	if err != nil {
		a.editFailure(w, r, destinationName, "The entry could not be added to the destination file.", http.StatusUnprocessableEntity)
		return
	}
	destinationCandidate, err := appendChange.ApplyToCueSource(destinationRaw)
	if err != nil {
		a.editFailure(w, r, destinationName, "The destination file changed before the entry could be added. Reload and try again.", http.StatusConflict)
		return
	}
	if _, err := cuebook.New(destinationCandidate); err != nil {
		a.editFailure(w, r, destinationName, "The entry does not satisfy the destination file's CUE constraints.", http.StatusUnprocessableEntity)
		return
	}

	deleteChange, err := patch.DeleteFromStructList(sourceRaw, sourceEntryValue)
	if err != nil {
		a.editFailure(w, r, sourceName, "The entry could not be removed from the source file.", http.StatusUnprocessableEntity)
		return
	}
	sourceCandidate, err := deleteChange.ApplyToCueSource(sourceRaw)
	if err != nil {
		a.editFailure(w, r, sourceName, "The source file changed before the entry could be removed. Reload and try again.", http.StatusConflict)
		return
	}
	if _, err := cuebook.New(sourceCandidate); err != nil {
		a.editFailure(w, r, sourceName, "Removing the entry does not satisfy the source file's CUE constraints.", http.StatusUnprocessableEntity)
		return
	}

	if err := a.committer.Commit(destinationName, patch.Validated(appendChange)); err != nil {
		status := http.StatusInternalServerError
		notice := "The entry could not be added to the destination file."
		if errors.Is(err, patch.ErrByteRangeNotFound) {
			status = http.StatusConflict
			notice = "The destination file changed before the entry could be added. Reload and try again."
		}
		a.editFailure(w, r, destinationName, notice, status)
		return
	}
	if err := a.committer.Commit(sourceName, patch.Validated(deleteChange)); err != nil {
		rollbackErr := a.committer.Commit(destinationName, patch.Validated(appendChange.Invert()))
		if rollbackErr != nil {
			a.editFailure(w, r, destinationName, "The entry was added to the destination, but the source could not be updated and the destination change could not be rolled back. It may now exist in both files.", http.StatusInternalServerError)
			return
		}
		status := http.StatusInternalServerError
		notice := "The transfer could not be completed; the destination change was rolled back."
		if errors.Is(err, patch.ErrByteRangeNotFound) {
			status = http.StatusConflict
			notice = "The source file changed before the entry could be removed. The destination change was rolled back; reload and try again."
		}
		a.editFailure(w, r, sourceName, notice, status)
		return
	}
	a.finishEdit(w, r, destinationName)
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

func (p swapSequencePatch) Difference() patch.ByteAnchor {
	if len(p.steps) == 0 {
		return patch.ByteAnchor{}
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
