package htmx

import (
	"context"
	"errors"
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

type itemRouteRequest struct {
	Path string `schema:"path"`
	File string `schema:"file"`
	Head string `schema:"head"`
	Tail string `schema:"tail"`
}

func (*itemRouteRequest) Validate(context.Context) error { return nil }

type itemResponse struct {
	pageData
	entryView
}

func (a *handler) item(_ context.Context, input *itemRouteRequest) (itemResponse, error) {
	filePath := input.Path
	if filePath == "" {
		filePath = input.File
	}
	head, headErr := strconv.Atoi(input.Head)
	tail, tailErr := strconv.Atoi(input.Tail)
	if headErr != nil || tailErr != nil || head < 0 || tail <= head || filePath == "" {
		return itemResponse{}, errors.New("The item request is invalid.")
	}
	requestedRange := cuebook.ByteRange{Head: head, Tail: tail}

	fileNames, err := a.fileNames()
	if err != nil {
		return itemResponse{}, errors.New("Unable to list CUE files.")
	}
	raw, document, status, message := a.readDocument(filePath, fileNames)
	if status != http.StatusOK {
		return itemResponse{}, documentReadError(filePath, status, message)
	}

	var selected *entryView
	index := 0
	for entry, err := range document.EachEntry() {
		if err != nil {
			return itemResponse{}, errors.New("Unable to display this CUE document.")
		}
		entryRange, err := cuebook.NewByteRange(entry.Value)
		if err != nil {
			return itemResponse{}, errors.New("Unable to locate this item in the CUE file.")
		}
		if entryRange == requestedRange {
			view := makeEntryView(entry, filePath, index, entryRange, a.committer == nil)
			selected = &view
			break
		}
		index++
	}
	if selected == nil {
		return itemResponse{}, &cuebook.ItemNotFoundError{Path: filePath, ByteRange: requestedRange}
	}

	page := a.basePage(fileNames, filePath, "")
	page.Selected = filePath
	setFileFrontmatter(&page, filePath, raw)
	page.SelectedEntry = selected

	return itemResponse{pageData: page, entryView: *selected}, nil
}
