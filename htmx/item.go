package htmx

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

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

func itemFailure(status int, message string) (itemResponse, error) {
	response := itemResponse{statusCode: status, message: message}
	return responseResult(response, status)
}

func (a *handler) item(_ context.Context, input *itemRouteRequest) (itemResponse, error) {
	filePath := input.Path
	if filePath == "" {
		filePath = input.File
	}
	head, headErr := strconv.Atoi(input.Head)
	tail, tailErr := strconv.Atoi(input.Tail)
	if headErr != nil || tailErr != nil || head < 0 || tail <= head || filePath == "" {
		return itemFailure(http.StatusBadRequest, "The item request is invalid.")
	}
	requestedRange := cuebook.ByteRange{Head: head, Tail: tail}

	fileNames, err := a.fileNames()
	if err != nil {
		return itemFailure(http.StatusInternalServerError, "Unable to list CUE files.")
	}
	_, document, status, message := a.readDocument(filePath, fileNames)
	if status != http.StatusOK {
		return itemFailure(status, message)
	}

	var selected *entryView
	index := 0
	for entry, err := range document.EachEntry() {
		if err != nil {
			return itemFailure(http.StatusUnprocessableEntity, "Unable to display this CUE document.")
		}
		entryRange, err := cuebook.NewByteRange(entry.Value)
		if err != nil {
			return itemFailure(http.StatusUnprocessableEntity, "Unable to locate this item in the CUE file.")
		}
		if entryRange == requestedRange {
			view := makeEntryView(entry, filePath, index, entryRange, a.committer == nil)
			selected = &view
			break
		}
		index++
	}
	if selected == nil {
		return itemFailure(http.StatusNotFound, "404 page not found")
	}

	response := itemResponse{entryView: *selected, statusCode: http.StatusOK}
	return responseResult(response, response.statusCode)
}
