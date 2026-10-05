package htmx

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

const archiveDirectory = ".archive/"

func (a *handler) delete(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(r) {
		a.renderPage(w, r, pageData{ReadOnly: a.committer == nil, Error: "Cross-origin edits are not allowed."}, http.StatusForbidden)
		return
	}
	if a.committer == nil {
		a.editFailure(w, r, "", "This source is read-only.", http.StatusForbidden)
		return
	}

	creator, ok := a.committer.(FileCreator)
	if !ok {
		a.editFailure(w, r, "", "This source cannot create archive files.", http.StatusNotImplemented)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := r.ParseForm(); err != nil {
		a.editFailure(w, r, "", "The delete request is invalid.", http.StatusBadRequest)
		return
	}

	fileName := r.PostForm.Get("file")
	if strings.HasPrefix(fileName, archiveDirectory) {
		a.editFailure(w, r, fileName, "Entries in the archive cannot be deleted.", http.StatusForbidden)
		return
	}

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

	from, err := strconv.Atoi(r.PostForm.Get("entry"))
	length, lengthErr := document.Len()
	if err != nil || lengthErr != nil || from < 0 || from >= length {
		a.editFailure(w, r, fileName, "The entry position is invalid.", http.StatusBadRequest)
		return
	}

	archiveName := archiveDirectory + time.Now().Format("2006-01-02") + ".cue"
	if err := creator.CreateFileIfNotExists(archiveName, []byte("[]\n")); err != nil {
		a.editFailure(w, r, fileName, "Unable to create the archive file.", http.StatusInternalServerError)
		return
	}

	fileNames, err = a.fileNames()
	if err != nil {
		a.editFailure(w, r, fileName, "Unable to list CUE files.", http.StatusInternalServerError)
		return
	}
	if !containsFile(fileNames, archiveName) {
		a.editFailure(w, r, fileName, "The archive file is not available in this source.", http.StatusInternalServerError)
		return
	}

	a.transferEntry(w, r, fileName, archiveName, raw, document, from, fileNames)
}
