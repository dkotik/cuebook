package htmx

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
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

func (a *handler) buildSearchIndex() {
	defer a.searchIndexReady.Store(true)

	fileNames, err := a.fileNames()
	if err != nil {
		a.searchIndexErr = fmt.Errorf("list CUE files for search index: %w", err)
		return
	}
	for _, filePath := range fileNames {
		_, document, status, message := a.readDocument(filePath, fileNames)
		if status != http.StatusOK {
			slog.Debug("skipping CUE file while building search index", "path", filePath, "status", status, "error", message)
			continue
		}
		for entry, err := range document.EachEntry() {
			if err != nil {
				slog.Warn("skipping CUE entry while building search index", "path", filePath, "error", err)
				continue
			}
			if err := a.searchIndex.Include(filePath, entry); err != nil {
				slog.Warn("unable to index CUE entry", "path", filePath, "error", err)
			}
		}
	}
}

func (a *handler) search(w http.ResponseWriter, r *http.Request) {
	if !a.searchIndexReady.Load() {
		w.Header().Set("Retry-After", "1")
		http.Error(w, "The search index is still being built.", http.StatusServiceUnavailable)
		return
	}
	if a.searchIndexErr != nil {
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

	results, err := a.searchIndex.Query(query)
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
