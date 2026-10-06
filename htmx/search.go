package htmx

import (
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

func (a *handler) searchClearButton(w http.ResponseWriter, r *http.Request) {
	var output strings.Builder
	if strings.TrimSpace(r.URL.Query().Get("q")) != "" {
		if err := a.templates.ExecuteTemplate(&output, "search-clear-button", nil); err != nil {
			http.Error(w, "Unable to render the clear search button.", http.StatusInternalServerError)
			return
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(output.String()))
}

func (a *handler) search(w http.ResponseWriter, r *http.Request) {
	if a.searchFS == nil || !a.searchFS.IndexReady() {
		w.Header().Set("Retry-After", "1")
		http.Error(w, "The search index is still being built.", http.StatusServiceUnavailable)
		return
	}
	if a.searchFS.IndexError() != nil {
		http.Error(w, "Unable to build the search index.", http.StatusInternalServerError)
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		query = strings.TrimSpace(r.URL.Query().Get("query"))
	}
	if query == "" {
		http.Error(w, "A search query is required.", http.StatusBadRequest)
		return
	}

	results, err := a.searchFS.Query(query)
	if err != nil {
		http.Error(w, "The search query is invalid.", http.StatusBadRequest)
		return
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

	var output strings.Builder
	if err := a.templates.ExecuteTemplate(&output, "search-results", view); err != nil {
		http.Error(w, "Unable to render search results.", http.StatusInternalServerError)
		return
	}
	w.Header().Add("Vary", "HX-Request")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(output.String()))
}
