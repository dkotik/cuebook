package htmx

import (
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
)

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
			wantStatus: http.StatusServiceUnavailable,
			wantBody:   "The search index is still being built.",
		},
		{
			name:       "missing query after indexing",
			ready:      true,
			path:       "/search",
			wantStatus: http.StatusBadRequest,
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
			response := httptest.NewRecorder()
			app.search(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", response.Code, test.wantStatus, response.Body.String())
			}
			if !strings.Contains(response.Body.String(), test.wantBody) {
				t.Errorf("body does not contain %q: %s", test.wantBody, response.Body.String())
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
		if response.Code != http.StatusServiceUnavailable {
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
	for _, want := range []string{"Needle entry", filePath} {
		if !strings.Contains(body, html.EscapeString(want)) {
			t.Errorf("search results do not contain %q: %s", want, body)
		}
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
