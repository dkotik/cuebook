package htmx

import (
	"context"
	"html"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/patch"
	"github.com/dkotik/cuebook/search"
	"github.com/dkotik/htadaptor"
)

func TestSearchFormIsInHeaderAndTargetsMainContent(t *testing.T) {
	handler, err := New(fstest.MapFS{
		"people.cue": &fstest.MapFile{Data: []byte(`[{Name: "Ada"}]`)},
	})
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}

	body := response.Body.String()
	headerEnd := strings.Index(body, "</header>")
	formStart := strings.Index(body, `<form class="topbar-search`)
	if formStart < 0 || headerEnd < 0 || formStart > headerEnd {
		t.Fatalf("search form is not inside the page header: %s", body)
	}
	navStart := strings.Index(body, `<nav class="file-nav`)
	navEnd := strings.Index(body, "</nav>")
	if navStart >= 0 && navEnd > navStart && formStart > navStart && formStart < navEnd {
		t.Errorf("search form is still inside the file navigation: %s", body)
	}
	for _, want := range []string{
		`hx-target="#workspace"`,
		`<main id="workspace"`,
		`hx-get="/search/clear-button"`,
		`hx-target="#search-clear-control"`,
		`id="search-clear-control"`,
		`aria-label="Search">🔍</button>`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("page does not contain %q", want)
		}
	}
	if strings.Contains(body, `id="search-results"`) {
		t.Errorf("page still has a separate search results target: %s", body)
	}
	if strings.Contains(body, `type="reset" aria-label="Clear search"`) {
		t.Errorf("clear search button is visible before a query is entered: %s", body)
	}
}

func TestSearchClearButtonTracksQuery(t *testing.T) {
	handler, err := New(fstest.MapFS{
		"people.cue": &fstest.MapFile{Data: []byte(`[{Name: "Ada"}]`)},
	})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		query      string
		wantButton bool
	}{
		{name: "empty query hides clear button"},
		{name: "whitespace query hides clear button", query: " \t"},
		{name: "nonempty query shows clear button", query: "Ada", wantButton: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			values := url.Values{"q": {test.query}}
			request := httptest.NewRequest(http.MethodGet, "http://example.test/search/clear-button?"+values.Encode(), nil)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
			}
			gotButton := strings.Contains(response.Body.String(), `type="reset" aria-label="Clear search"`)
			if gotButton != test.wantButton {
				t.Errorf("clear button visible = %t, want %t; body: %s", gotButton, test.wantButton, response.Body.String())
			}
		})
	}
}

func TestSearchHandlerChecksReadinessAndQuery(t *testing.T) {
	tests := []struct {
		name       string
		ready      bool
		path       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "index is still building",
			path:       "/search?q=needle",
			wantStatus: http.StatusInternalServerError,
			wantBody:   "The search index is still being built.",
		},
		{
			name:       "missing query after indexing",
			ready:      true,
			path:       "/search",
			wantStatus: http.StatusInternalServerError,
			wantBody:   "A search query is required.",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			app := &handler{}
			if test.ready {
				app.searchFS = completedSearchFS{FS: fstest.MapFS{}}
			}
			request := httptest.NewRequest(http.MethodGet, "http://example.test"+test.path, nil)
			_, err := app.search(context.Background(), &searchRequest{
				Query: request.URL.Query().Get("q"),
				Alt:   request.URL.Query().Get("query"),
			})
			if test.wantStatus == http.StatusOK {
				if err != nil {
					t.Fatal(err)
				}
			} else if status := htadaptor.GetHyperTextStatusCode(err); status != test.wantStatus {
				t.Fatalf("error status = %d, want %d; error: %v", status, test.wantStatus, err)
			}
			if err == nil || !strings.Contains(err.Error(), test.wantBody) {
				t.Errorf("error does not contain %q: %v", test.wantBody, err)
			}
		})
	}
}

type completedSearchFS struct {
	fs.FS
}

func (completedSearchFS) IndexReady() bool  { return true }
func (completedSearchFS) IndexError() error { return nil }
func (completedSearchFS) Query(string) ([]search.Result, error) {
	return nil, nil
}
func (completedSearchFS) UpdateFile(string) error                    { return nil }
func (completedSearchFS) ApplyFileChange(string, func() error) error { return nil }
func (completedSearchFS) RemoveFile(string) error                    { return nil }

func TestSearchIndexRefreshesAfterCommittedEdit(t *testing.T) {
	t.Parallel()

	const filePath = "people.cue"
	files := fstest.MapFS{
		filePath: {
			Data: []byte(`[{Name: "Target entry", Text: "oldtoken"}]`),
		},
	}
	handler, err := NewWithCommitter(files, mapFileCommitter{files: files})
	if err != nil {
		t.Fatal(err)
	}
	waitForSearchResponse(t, handler, "oldtoken")

	form := url.Values{
		"file":  {filePath},
		"entry": {"0"},
		"field": {"Text"},
		"value": {"newtoken"},
	}
	request := httptest.NewRequest(http.MethodPost, "http://example.test/edit", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Origin", "http://example.test")
	request.Header.Set("HX-Request", "true")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("edit status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}

	oldResults := issueSearchRequest(t, handler, "oldtoken")
	if strings.Contains(oldResults.Body.String(), "Target entry") {
		t.Errorf("stale search result remained after edit: %s", oldResults.Body.String())
	}
	newResults := issueSearchRequest(t, handler, "newtoken")
	if !strings.Contains(newResults.Body.String(), "Target entry") {
		t.Errorf("updated search result missing after edit: %s", newResults.Body.String())
	}
}

type mapFileCommitter struct {
	files fstest.MapFS
}

func (c mapFileCommitter) Commit(name string, change patch.Patch) error {
	file, ok := c.files[name]
	if !ok {
		return fs.ErrNotExist
	}
	updated, err := change.ApplyToCueSource(file.Data)
	if err != nil {
		return err
	}
	file.Data = updated
	return nil
}

func waitForSearchResponse(t *testing.T, handler http.Handler, query string) *httptest.ResponseRecorder {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		response := issueSearchRequest(t, handler, query)
		if !strings.Contains(response.Body.String(), "The search index is still being built.") {
			return response
		}
		if time.Now().After(deadline) {
			t.Fatal("search index did not finish building")
		}
		time.Sleep(time.Millisecond)
	}
}

func issueSearchRequest(t *testing.T, handler http.Handler, query string) *httptest.ResponseRecorder {
	t.Helper()
	values := url.Values{}
	values.Set("q", query)
	request := httptest.NewRequest(http.MethodGet, "http://example.test/search?"+values.Encode(), nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func TestSearchResultsExposeDraggableSourceEntries(t *testing.T) {
	tests := []struct {
		name      string
		writable  bool
		wantMoves bool
	}{
		{name: "writable source", writable: true, wantMoves: true},
		{name: "read-only source"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			files := fstest.MapFS{
				"first.cue":  &fstest.MapFile{Data: []byte(`[{Name: "First untouched", Text: "other"}, {Name: "First needle", Text: "needle"}]`)},
				"second.cue": &fstest.MapFile{Data: []byte(`[{Name: "Second untouched", Text: "other"}, {Name: "Second needle", Text: "needle"}]`)},
			}

			var handler http.Handler
			var err error
			if test.writable {
				handler, err = NewWithCommitter(files, mapFileCommitter{files: files})
			} else {
				handler, err = New(files)
			}
			if err != nil {
				t.Fatal(err)
			}

			response := waitForSearchResponse(t, handler, "needle")
			if response.Code != http.StatusOK {
				t.Fatalf("search status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
			}
			body := response.Body.String()
			if got := strings.Count(body, `<article class="entry card cell"`); got != 2 {
				t.Fatalf("search result cards = %d, want 2: %s", got, body)
			}

			gotMoves := strings.Contains(body, `data-entry-drag-handle`)
			if gotMoves != test.wantMoves {
				t.Fatalf("search result drag handles visible = %t, want %t: %s", gotMoves, test.wantMoves, body)
			}
			if test.wantMoves {
				for _, want := range []string{
					`data-entry-index="1" data-file="first.cue" data-entry-count="2"`,
					`data-entry-index="1" data-file="second.cue" data-entry-count="2"`,
					`draggable="true" data-entry-drag-handle`,
				} {
					if !strings.Contains(body, want) {
						t.Errorf("search results do not contain %q: %s", want, body)
					}
				}
			}
		})
	}
}

func TestSearchResultsLinkToMatchingItem(t *testing.T) {
	t.Parallel()

	const filePath = "nested/people & contacts.cue"
	source := []byte(`[
	{Name: "Needle entry", Email: "needle@example.test"},
	{Name: "Other entry", Email: "other@example.test"},
]`)
	book, err := cuebook.New(source)
	if err != nil {
		t.Fatal(err)
	}
	var expectedRange cuebook.ByteRange
	for entry, err := range book.EachEntry() {
		if err != nil {
			t.Fatal(err)
		}
		expectedRange, err = cuebook.NewByteRange(entry.Value)
		if err != nil {
			t.Fatal(err)
		}
		break
	}

	handler, err := New(fstest.MapFS{
		filePath: &fstest.MapFile{Data: source},
	})
	if err != nil {
		t.Fatal(err)
	}
	searchResponse := waitForSearchResponse(t, handler, "needle")
	if searchResponse.Code != http.StatusOK {
		t.Fatalf("search status = %d, want %d; body: %s", searchResponse.Code, http.StatusOK, searchResponse.Body.String())
	}

	body := searchResponse.Body.String()
	for _, want := range []string{"Needle entry", filePath, "needle@example.test"} {
		if !strings.Contains(body, html.EscapeString(want)) {
			t.Errorf("search results do not contain %q: %s", want, body)
		}
	}
	for _, want := range []string{
		`<article class="entry card cell"`,
		`class="card-content search-result-description"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("search results do not contain %q: %s", want, body)
		}
	}
	cardStart := strings.Index(body, `<article class="entry card cell"`)
	if cardStart < 0 {
		t.Fatalf("search result card is missing: %s", body)
	}
	headerEnd := strings.Index(body[cardStart:], "</header>")
	if headerEnd < 0 {
		t.Fatalf("search result card header is missing: %s", body)
	}
	pathContent := strings.TrimLeft(body[cardStart+headerEnd+len("</header>"):], "\t\r\n ")
	if !strings.HasPrefix(pathContent, `<p class="search-result-path muted">`) {
		t.Fatalf("source path does not immediately follow the result header: %s", body)
	}
	if !strings.Contains(pathContent, html.EscapeString(filePath)) {
		t.Errorf("source path is missing beneath result header: %s", pathContent)
	}
	if got := strings.Count(body, html.EscapeString("Needle entry")); got != 1 {
		t.Errorf("result title appears %d times, want once", got)
	}
	hrefAttributeStart := strings.Index(body, `href="`)
	if hrefAttributeStart < 0 {
		t.Fatalf("search result has no item link: %s", body)
	}
	hrefStart := hrefAttributeStart + len(`href="`)
	hrefLength := strings.Index(body[hrefStart:], `"`)
	if hrefLength < 0 {
		t.Fatalf("search result link href is unterminated: %s", body)
	}
	itemHref := html.UnescapeString(body[hrefStart : hrefStart+hrefLength])
	parsedHref, err := url.Parse(itemHref)
	if err != nil {
		t.Fatalf("parse search result href %q: %v", itemHref, err)
	}
	if parsedHref.Path != "/item" {
		t.Errorf("search result path = %q, want /item", parsedHref.Path)
	}
	query := parsedHref.Query()
	if got := query.Get("path"); got != filePath {
		t.Errorf("item path query = %q, want %q", got, filePath)
	}
	if got := query.Get("head"); got != strconv.Itoa(expectedRange.Head) {
		t.Errorf("item head query = %q, want %d", got, expectedRange.Head)
	}
	if got := query.Get("tail"); got != strconv.Itoa(expectedRange.Tail) {
		t.Errorf("item tail query = %q, want %d", got, expectedRange.Tail)
	}

	itemRequest := httptest.NewRequest(http.MethodGet, "http://example.test"+itemHref, nil)
	itemResponse := httptest.NewRecorder()
	handler.ServeHTTP(itemResponse, itemRequest)
	if itemResponse.Code != http.StatusOK {
		t.Fatalf("item status = %d, want %d; body: %s", itemResponse.Code, http.StatusOK, itemResponse.Body.String())
	}
	itemBody := itemResponse.Body.String()
	for _, want := range []string{"Needle entry", "needle@example.test"} {
		if !strings.Contains(itemBody, want) {
			t.Errorf("item body does not contain %q: %s", want, itemBody)
		}
	}
	for _, unwanted := range []string{"Other entry", "other@example.test"} {
		if strings.Contains(itemBody, unwanted) {
			t.Errorf("item body unexpectedly contains %q: %s", unwanted, itemBody)
		}
	}
}
