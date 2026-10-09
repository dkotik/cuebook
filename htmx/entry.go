package htmx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/format"
	"github.com/dkotik/cuebook"
)

type EntryRequest struct {
	cuebook.ByteRange
	Path string
}

func entryRequestFromQuery(query url.Values) (EntryRequest, error) {
	head, err := strconv.Atoi(query.Get("head"))
	if err != nil {
		return EntryRequest{}, fmt.Errorf("invalid entry range head: %w", err)
	}
	tail, err := strconv.Atoi(query.Get("tail"))
	if err != nil {
		return EntryRequest{}, fmt.Errorf("invalid entry range tail: %w", err)
	}
	if head < 0 || tail <= head {
		return EntryRequest{}, fmt.Errorf("invalid entry byte range")
	}

	filePath := query.Get("path")
	if filePath == "" {
		filePath = query.Get("file")
	}
	if filePath == "" {
		return EntryRequest{}, fmt.Errorf("file path is required")
	}

	return EntryRequest{
		ByteRange: cuebook.ByteRange{Head: head, Tail: tail},
		Path:      filePath,
	}, nil
}

type entryRouteRequest struct {
	Path string `schema:"path"`
	File string `schema:"file"`
	Head string `schema:"head"`
	Tail string `schema:"tail"`
}

func (*entryRouteRequest) Validate(context.Context) error { return nil }

type entryResponse struct {
	pageData
	entryView
}

func (a *handler) entry(_ context.Context, input *entryRouteRequest) (entryResponse, error) {
	filePath := input.Path
	if filePath == "" {
		filePath = input.File
	}
	head, headErr := strconv.Atoi(input.Head)
	tail, tailErr := strconv.Atoi(input.Tail)
	if headErr != nil || tailErr != nil || head < 0 || tail <= head || filePath == "" {
		return entryResponse{}, errors.New("The entry request is invalid.")
	}
	return a.loadEntryPage(filePath, cuebook.ByteRange{Head: head, Tail: tail}, -1)
}

func (a *handler) loadEntryPage(filePath string, requestedRange cuebook.ByteRange, requestedIndex int) (entryResponse, error) {
	fileNames, err := a.fileNames()
	if err != nil {
		return entryResponse{}, errors.New("Unable to list CUE files.")
	}
	raw, document, status, message := a.readDocument(filePath, fileNames)
	if status != http.StatusOK {
		return entryResponse{}, documentReadError(filePath, status, message)
	}

	var selected *entryView
	index := 0
	for entry, err := range document.EachEntry() {
		if err != nil {
			return entryResponse{}, errors.New("Unable to display this CUE document.")
		}
		entryRange, err := cuebook.NewByteRange(entry.Value)
		if err != nil {
			return entryResponse{}, errors.New("Unable to locate this entry in the CUE file.")
		}
		if (requestedIndex >= 0 && index == requestedIndex) || (requestedIndex < 0 && entryRange == requestedRange) {
			view := makeEntryView(entry, filePath, index, entryRange, a.committer == nil, true)
			source, err := entryCUESource(entry.Value)
			if err != nil {
				return entryResponse{}, fmt.Errorf("Unable to format this entry as CUE: %w", err)
			}
			view.CUESource = source
			view.CutFingerprint = entrySourceFingerprint(source)
			if view.CanMove {
				for _, name := range fileNames {
					view.MoveFiles = append(view.MoveFiles, moveFileView{Name: name, Current: name == filePath})
				}
			}
			selected = &view
			break
		}
		index++
	}
	if selected == nil {
		return entryResponse{}, &cuebook.EntryNotFoundError{Path: filePath, ByteRange: requestedRange}
	}

	page := a.basePage(fileNames, filePath, "")
	page.Selected = filePath
	setFileFrontmatter(&page, filePath, raw)
	page.SelectedEntry = selected

	return entryResponse{pageData: page, entryView: *selected}, nil
}

func entryCUESource(value cue.Value) (string, error) {
	source, err := format.Node(value.Syntax(cue.Final(), cue.Concrete(true)), format.Simplify())
	return string(source), err
}
