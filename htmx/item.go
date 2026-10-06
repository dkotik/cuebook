package htmx

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/dkotik/cuebook"
)

type ItemRequest struct {
	cuebook.ByteRange
	Path string
}

func itemRequestFromQuery(query url.Values) (ItemRequest, error) {
	head, err := strconv.Atoi(query.Get("head"))
	if err != nil {
		return ItemRequest{}, fmt.Errorf("invalid item range head: %w", err)
	}
	tail, err := strconv.Atoi(query.Get("tail"))
	if err != nil {
		return ItemRequest{}, fmt.Errorf("invalid item range tail: %w", err)
	}
	if head < 0 || tail <= head {
		return ItemRequest{}, fmt.Errorf("invalid item byte range")
	}

	filePath := query.Get("path")
	if filePath == "" {
		filePath = query.Get("file")
	}
	if filePath == "" {
		return ItemRequest{}, fmt.Errorf("file path is required")
	}

	return ItemRequest{
		ByteRange: cuebook.ByteRange{Head: head, Tail: tail},
		Path:      filePath,
	}, nil
}

func (a *handler) item(w http.ResponseWriter, r *http.Request) {
	request, err := itemRequestFromQuery(r.URL.Query())
	if err != nil {
		http.Error(w, "The item request is invalid.", http.StatusBadRequest)
		return
	}

	fileNames, err := a.fileNames()
	if err != nil {
		http.Error(w, "Unable to list CUE files.", http.StatusInternalServerError)
		return
	}
	_, document, status, message := a.readDocument(request.Path, fileNames)
	if status != http.StatusOK {
		http.Error(w, message, status)
		return
	}

	var selected *entryView
	index := 0
	for entry, err := range document.EachEntry() {
		if err != nil {
			http.Error(w, "Unable to display this CUE document.", http.StatusUnprocessableEntity)
			return
		}
		entryRange, err := cuebook.NewByteRange(entry.Value)
		if err != nil {
			http.Error(w, "Unable to locate this item in the CUE file.", http.StatusUnprocessableEntity)
			return
		}
		if entryRange == request.ByteRange {
			view := makeEntryView(entry, request.Path, index, entryRange, a.committer == nil)
			selected = &view
			break
		}
		index++
	}
	if selected == nil {
		http.NotFound(w, r)
		return
	}

	var output strings.Builder
	if err := a.templates.ExecuteTemplate(&output, "entry-item", *selected); err != nil {
		http.Error(w, "Unable to render the item.", http.StatusInternalServerError)
		return
	}
	w.Header().Add("Vary", "HX-Request")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(output.String()))
}
