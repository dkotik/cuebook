package htmx

import (
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/dkotik/cuebook"
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
			app.searchIndexReady.Store(test.ready)
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
	searchQuery := url.Values{}
	searchQuery.Set("q", "needle")
	searchPath := "/search?" + searchQuery.Encode()

	deadline := time.Now().Add(5 * time.Second)
	var searchResponse *httptest.ResponseRecorder
	for {
		request := httptest.NewRequest(http.MethodGet, "http://example.test"+searchPath, nil)
		searchResponse = httptest.NewRecorder()
		handler.ServeHTTP(searchResponse, request)
		if searchResponse.Code != http.StatusServiceUnavailable {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("search index did not finish building")
		}
		time.Sleep(time.Millisecond)
	}
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
