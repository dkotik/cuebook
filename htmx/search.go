package htmx

import (
	"context"
	"errors"
	"io/fs"
	"strings"

	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/patch"
)

type searchResultsView struct {
	Query   string
	Results []searchResultView
}

type searchResultView struct {
	Index       int
	File        string
	ItemURL     string
	Title       string
	CanMove     bool
	EntryCount  int
	Path        string
	Description []string
}

type searchEntryPosition struct {
	Index int
	Count int
}

func (a *handler) searchEntryPositions(filePath string) (map[cuebook.ByteRange]searchEntryPosition, error) {
	source, err := fs.ReadFile(a.searchFS, filePath)
	if err != nil {
		return nil, err
	}
	document, err := cuebook.New(source)
	if err != nil {
		return nil, err
	}
	count, err := document.Len()
	if err != nil {
		return nil, err
	}

	positions := make(map[cuebook.ByteRange]searchEntryPosition, count)
	index := 0
	for entry, err := range document.EachEntry() {
		if err != nil {
			return nil, err
		}
		byteRange, err := cuebook.NewByteRange(entry.Value)
		if err != nil {
			return nil, err
		}
		positions[byteRange] = searchEntryPosition{Index: index, Count: count}
		index++
	}
	return positions, nil
}

func (a *handler) applyFileChange(filePath string, change func() error) error {
	if a.searchFS == nil {
		return change()
	}
	return a.searchFS.ApplyFileChange(filePath, change)
}

func (a *handler) commitFile(filePath string, change patch.Patch) error {
	return a.applyFileChange(filePath, func() error {
		return a.committer.Commit(filePath, change)
	})
}

type searchClearRequest struct {
	Query string `schema:"q"`
}

func (*searchClearRequest) Validate(context.Context) error { return nil }

type searchClearButtonResponse struct {
	Visible bool
}

func (a *handler) searchClearButton(_ context.Context, request *searchClearRequest) (searchClearButtonResponse, error) {
	return searchClearButtonResponse{
		Visible: strings.TrimSpace(request.Query) != "",
	}, nil
}

type searchResponse struct {
	searchResultsView
}

type searchRequest struct {
	Query string `schema:"q"`
	Alt   string `schema:"query"`
}

func (*searchRequest) Validate(context.Context) error { return nil }

func (a *handler) search(_ context.Context, request *searchRequest) (searchResponse, error) {
	if a.searchFS == nil || !a.searchFS.IndexReady() {
		return searchResponse{}, errors.New("The search index is still being built.")
	}
	if a.searchFS.IndexError() != nil {
		return searchResponse{}, errors.New("Unable to build the search index.")
	}

	query := strings.TrimSpace(request.Query)
	if query == "" {
		query = strings.TrimSpace(request.Alt)
	}
	if query == "" {
		return searchResponse{}, errors.New("A search query is required.")
	}

	results, err := a.searchFS.Query(query)
	if err != nil {
		return searchResponse{}, errors.New("The search query is invalid.")
	}
	view := searchResultsView{Query: query, Results: make([]searchResultView, 0, len(results))}
	positionsByFile := make(map[string]map[cuebook.ByteRange]searchEntryPosition)
	for _, result := range results {
		title := result.Entry.GetTitle()
		if title == "" {
			title = "Untitled entry"
		}
		resultView := searchResultView{
			File:        result.Path,
			ItemURL:     itemURL(result.Path, result.ByteRange),
			Title:       title,
			Path:        result.Path,
			Description: result.Entry.GetDescription(),
		}
		if a.committer != nil {
			positions, exists := positionsByFile[result.Path]
			if !exists {
				positions, _ = a.searchEntryPositions(result.Path)
				positionsByFile[result.Path] = positions
			}
			if position, ok := positions[result.ByteRange]; ok {
				resultView.Index = position.Index
				resultView.EntryCount = position.Count
				resultView.CanMove = true
			}
		}
		view.Results = append(view.Results, resultView)
	}

	response := searchResponse{searchResultsView: view}
	return response, nil
}
