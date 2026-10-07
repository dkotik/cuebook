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
	message    string
	retryAfter string
}

func searchFailure(status int, message, retryAfter string) (searchResponse, error) {
	response := searchResponse{message: message, retryAfter: retryAfter}
	return responseResult(response, responseErrorForStatus(status))
}

type searchRequest struct {
	Query string `schema:"q"`
	Alt   string `schema:"query"`
}

func (*searchRequest) Validate(context.Context) error { return nil }

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

	response := searchResponse{searchResultsView: view}
	return response, nil
}
