package htmx

import (
	"context"
	"errors"
	"io/fs"
	"strings"

	"github.com/dkotik/cuebook"
)

type searchResultsView struct {
	Query   string
	Results []searchResultView
}

type searchResultView struct {
	Index       int
	File        string
	EntryURL    string
	Title       string
	CanMove     bool
	EntryCount  int
	Path        string
	Description []string
}

func searchEntryDescription(entry cuebook.Entry) []string {
	description := entry.GetDescription()
	for i := range description {
		field, err := entry.GetField(i)
		if err != nil {
			continue
		}
		view := makeFieldView(field, "", 0, true, false)
		if !view.Boolean {
			continue
		}
		if view.Secret {
			if !view.Empty {
				description[i] = "••••••••"
			}
			continue
		}
		var symbol string
		switch view.Value {
		case "true":
			symbol = "✔"
		case "false":
			symbol = "✖"
		case "not_set":
			symbol = "☐"
		default:
			continue
		}
		description[i] = symbol + "\u00a0" + view.Name
	}
	return description
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

func (a *handler) commitFile(ctx context.Context, filePath string, change cuebook.Patch) error {
	return a.applyFileChange(filePath, func() error {
		_, err := cuebook.Commit(ctx, a.files, filePath, change)
		return err
	})
}

// createFileIfNotExists writes content to filePath only when the file is
// missing. applyFileChange serializes it with every other change made by this
// handler, so a concurrent request cannot create the file in between.
func (a *handler) createFileIfNotExists(ctx context.Context, filePath string, content []byte) error {
	return a.applyFileChange(filePath, func() error {
		_, err := a.files.ReadFile(ctx, filePath)
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		return a.files.WriteFile(ctx, filePath, content)
	})
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
		return searchResponse{}, ErrSearchIndexBuilding
	}
	if err := a.searchFS.IndexError(); err != nil {
		return searchResponse{}, CauseError{Code: ErrUnableToBuildSearchIndex, Cause: err}
	}

	query := strings.TrimSpace(request.Query)
	if query == "" {
		query = strings.TrimSpace(request.Alt)
	}
	if query == "" {
		return searchResponse{}, ErrSearchQueryRequired
	}

	results, err := a.searchFS.Query(query)
	if err != nil {
		return searchResponse{}, CauseError{Code: ErrSearchQueryInvalid, Cause: err}
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
			EntryURL:    entryURL(result.Path, result.ByteRange),
			Title:       title,
			Path:        result.Path,
			Description: searchEntryDescription(result.Entry),
		}
		if a.files != nil {
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
