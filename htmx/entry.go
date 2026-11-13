package htmx

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/format"
	"github.com/dkotik/cuebook"
)

type entryRouteRequest struct {
	Path string `schema:"path"`
	File string `schema:"file"`
	Head string `schema:"head"`
	Tail string `schema:"tail"`
}

func (*entryRouteRequest) Validate(context.Context) error { return nil }

type entryUUIDRouteRequest struct {
	Path string `schema:"path"`
	File string `schema:"file"`
	UUID string `schema:"uuid"`
}

func (*entryUUIDRouteRequest) Validate(context.Context) error { return nil }

type entryResponse struct {
	pageData
	entryView
}

func (a *handler) entry(_ context.Context, input *entryRouteRequest) (entryResponse, error) {
	filePath := input.Path
	if filePath == "" {
		filePath = input.File
	}
	if filePath == "" {
		return entryResponse{}, ErrInvalidEntryRequest
	}
	head, headErr := strconv.Atoi(input.Head)
	tail, tailErr := strconv.Atoi(input.Tail)
	if headErr != nil || tailErr != nil || head < 0 || tail <= head {
		return entryResponse{}, ErrInvalidEntryRequest
	}
	return a.loadEntryPageByRange(filePath, cuebook.ByteRange{Head: head, Tail: tail})
}

func (a *handler) entryUUID(_ context.Context, input *entryUUIDRouteRequest) (entryResponse, error) {
	filePath := input.Path
	if filePath == "" {
		filePath = input.File
	}
	if filePath == "" || input.UUID == "" {
		return entryResponse{}, ErrInvalidEntryRequest
	}
	return a.loadEntryPageByUUID(filePath, input.UUID)
}

func (a *handler) loadEntryDocument(filePath string) ([]string, []byte, cuebook.Book, error) {
	fileNames, err := a.fileNames()
	if err != nil {
		return nil, nil, cuebook.Book{}, ErrUnableToListCueFiles
	}
	raw, document, err := a.readDocument(filePath, fileNames)
	if err != nil {
		return nil, nil, cuebook.Book{}, err
	}
	return fileNames, raw, document, nil
}

func (a *handler) loadEntryPageByRange(filePath string, requestedRange cuebook.ByteRange) (entryResponse, error) {
	fileNames, raw, document, err := a.loadEntryDocument(filePath)
	if err != nil {
		return entryResponse{}, err
	}

	index := 0
	for entry, entryErr := range document.EachEntry() {
		if entryErr != nil {
			return entryResponse{}, CauseError{Code: ErrUnableToDisplayCUEDocument, Cause: entryErr}
		}
		entryRange, rangeErr := cuebook.NewByteRange(entry.Value)
		if rangeErr != nil {
			return entryResponse{}, CauseError{Code: ErrUnableToLocateEntry, Cause: rangeErr}
		}
		if entryRange == requestedRange {
			return a.renderEntryPage(filePath, fileNames, raw, document, index)
		}
		index++
	}
	return entryResponse{}, &cuebook.EntryNotFoundError{Path: filePath, ByteRange: requestedRange}
}

func (a *handler) loadEntryPageByIndex(filePath string, index int) (entryResponse, error) {
	fileNames, raw, document, err := a.loadEntryDocument(filePath)
	if err != nil {
		return entryResponse{}, err
	}
	return a.renderEntryPage(filePath, fileNames, raw, document, index)
}

func (a *handler) loadEntryPageByUUID(filePath, uuid string) (entryResponse, error) {
	fileNames, raw, document, err := a.loadEntryDocument(filePath)
	if err != nil {
		return entryResponse{}, err
	}
	index, err := entryIndexByUUID(document, uuid)
	if errors.Is(err, cuebook.ErrBookEntryNotFound) {
		return entryResponse{}, &cuebook.EntryNotFoundError{Path: filePath}
	}
	if err != nil {
		return entryResponse{}, err
	}
	return a.renderEntryPage(filePath, fileNames, raw, document, index)
}

func (a *handler) renderEntryPage(filePath string, fileNames []string, raw []byte, document cuebook.Book, index int) (entryResponse, error) {
	entry, err := document.GetEntryByIndex(index)
	if err != nil {
		if errors.Is(err, cuebook.ErrBookEntryIndexOutOfRange) {
			return entryResponse{}, &cuebook.EntryNotFoundError{Path: filePath}
		}
		return entryResponse{}, CauseError{Code: ErrUnableToDisplayCUEDocument, Cause: err}
	}
	entryRange, err := cuebook.NewByteRange(entry.Value)
	if err != nil {
		return entryResponse{}, CauseError{Code: ErrUnableToLocateEntry, Cause: err}
	}
	view := makeEntryView(entry, filePath, index, entryRange, a.files == nil, true)
	source, err := entryCUESource(entry.Value)
	if err != nil {
		return entryResponse{}, fmt.Errorf("Unable to format this entry as CUE: %w", err)
	}
	view.CUESource = source
	view.CutFingerprint = entrySourceFingerprint(source)
	if view.CanAddField {
		view.NewFieldKinds = makeNewFieldKindViews()
	}
	if view.CanMove {
		view.MoveFilePicker = makeFilePicker(fileNames, filePath, filePickerOptions{
			Legend:         "Destination file",
			InputName:      "destination",
			DisableCurrent: true,
		})
	}

	page := a.basePage(fileNames, filePath, "")
	page.Selected = filePath
	setFileFrontmatter(&page, raw)
	page.FileBreadcrumb = makeFileBreadcrumb(filePath)
	page.FileBreadcrumb[len(page.FileBreadcrumb)-1].URL = fileURL(filePath)
	page.SelectedEntry = &view
	if !page.ReadOnly {
		page.AddFields = makeAddFieldViews(document)
		page.RequiredAddFields, page.OptionalAddFields = splitAddFieldViews(page.AddFields)
	}

	return entryResponse{pageData: page, entryView: view}, nil
}

func entryCUESource(value cue.Value) (string, error) {
	source, err := format.Node(value.Syntax(cue.Final(), cue.Concrete(true), cue.Attributes(false)), format.Simplify())
	return string(source), err
}
