package htmx

import (
	"context"
	"net/http"
	"strings"

	"github.com/dkotik/cuebook/patch"
)

type searchResultsView struct {
	Query   string
	Results []searchResultView
}

type searchResultView struct {
	Title string
	Path  string
	URL   string
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

func (a *handler) searchClearButton(_ context.Context, request *searchClearRequest) (searchClearButtonResponse, error) {
	return searchClearButtonResponse{
		Visible:    strings.TrimSpace(request.Query) != "",
		statusCode: http.StatusOK,
	}, nil
}

func searchFailure(status int, message, retryAfter string) (searchResponse, error) {
	response := searchResponse{statusCode: status, message: message, retryAfter: retryAfter}
	return responseResult(response, status)
}

func (a *handler) search(_ context.Context, request *searchRequest) (searchResponse, error) {
	if a.searchFS == nil || !a.searchFS.IndexReady() {
		return searchFailure(http.StatusServiceUnavailable, "The search index is still being built.", "1")
	}
	if a.searchFS.IndexError() != nil {
		return searchFailure(http.StatusInternalServerError, "Unable to build the search index.", "")
	}

	query := strings.TrimSpace(request.Query)
	if query == "" {
		query = strings.TrimSpace(request.Alt)
	}
	if query == "" {
		return searchFailure(http.StatusBadRequest, "A search query is required.", "")
	}

	results, err := a.searchFS.Query(query)
	if err != nil {
		return searchFailure(http.StatusBadRequest, "The search query is invalid.", "")
	}
	view := searchResultsView{Query: query, Results: make([]searchResultView, 0, len(results))}
	for _, result := range results {
		title := result.Entry.GetTitle()
		if title == "" {
			title = "Untitled entry"
		}
		view.Results = append(view.Results, searchResultView{
			Title: title,
			Path:  result.Path,
			URL:   itemURL(result.Path, result.ByteRange),
		})
	}

	response := searchResponse{searchResultsView: view, statusCode: http.StatusOK}
	return responseResult(response, response.statusCode)
}
