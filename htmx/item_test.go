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

	"github.com/dkotik/cuebook"
)

func TestIndexEntryLinksOpenMatchingItem(t *testing.T) {
	t.Parallel()

	source := []byte(`[
	{Name: "First entry", Email: "first@example.test"},
	{Name: "Second entry", Email: "second@example.test", Active: true},
]`)
	book, err := cuebook.New(source)
	if err != nil {
		t.Fatal(err)
	}
	var selectedRange cuebook.ByteRange
	index := 0
	for entry, err := range book.EachEntry() {
		if err != nil {
			t.Fatal(err)
		}
		if index == 1 {
			selectedRange, err = cuebook.NewByteRange(entry.Value)
			if err != nil {
				t.Fatal(err)
			}
			break
		}
		index++
	}

	tests := []struct {
		name     string
		filePath string
	}{
		{name: "nested file path", filePath: "nested/people.cue"},
		{name: "reserved characters in file path", filePath: "nested/people & groups.cue"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			handler, err := New(fstest.MapFS{
				test.filePath: &fstest.MapFile{Data: source},
			})
			if err != nil {
				t.Fatal(err)
			}

			indexQuery := url.Values{}
			indexQuery.Set("file", test.filePath)
			indexRequest := httptest.NewRequest(http.MethodGet, "http://example.test/?"+indexQuery.Encode(), nil)
			indexResponse := httptest.NewRecorder()
			handler.ServeHTTP(indexResponse, indexRequest)
			if indexResponse.Code != http.StatusOK {
				t.Fatalf("index status = %d, want %d; body: %s", indexResponse.Code, http.StatusOK, indexResponse.Body.String())
			}

			body := indexResponse.Body.String()
			titlePosition := strings.Index(body, "Second entry")
			if titlePosition < 0 {
				t.Fatalf("selected entry title not found in index: %s", body)
			}
			anchorStart := strings.LastIndex(body[:titlePosition], "<a ")
			if anchorStart < 0 {
				t.Fatalf("selected entry title is not linked: %s", body)
			}
			hrefAttributeStart := strings.Index(body[anchorStart:], `href="`)
			if hrefAttributeStart < 0 {
				t.Fatalf("selected entry link has no href: %s", body[anchorStart:])
			}
			hrefStart := anchorStart + hrefAttributeStart + len(`href="`)
			hrefLength := strings.Index(body[hrefStart:], `"`)
			if hrefLength < 0 {
				t.Fatalf("selected entry link href is unterminated: %s", body[anchorStart:])
			}
			itemHref := html.UnescapeString(body[hrefStart : hrefStart+hrefLength])
			parsedHref, err := url.Parse(itemHref)
			if err != nil {
				t.Fatalf("parse generated item href %q: %v", itemHref, err)
			}
			if parsedHref.Path != "/item" {
				t.Errorf("item href path = %q, want /item", parsedHref.Path)
			}
			query := parsedHref.Query()
			if got := query.Get("path"); got != test.filePath {
				t.Errorf("item href path query = %q, want %q", got, test.filePath)
			}
			if got := query.Get("head"); got != strconv.Itoa(selectedRange.Head) {
				t.Errorf("item href head query = %q, want %d", got, selectedRange.Head)
			}
			if got := query.Get("tail"); got != strconv.Itoa(selectedRange.Tail) {
				t.Errorf("item href tail query = %q, want %d", got, selectedRange.Tail)
			}

			itemRequest := httptest.NewRequest(http.MethodGet, "http://example.test"+itemHref, nil)
			itemResponse := httptest.NewRecorder()
			handler.ServeHTTP(itemResponse, itemRequest)
			if itemResponse.Code != http.StatusOK {
				t.Fatalf("item status = %d, want %d; body: %s", itemResponse.Code, http.StatusOK, itemResponse.Body.String())
			}
			itemBody := itemResponse.Body.String()
			for _, want := range []string{"Second entry", "second@example.test"} {
				if !strings.Contains(itemBody, want) {
					t.Errorf("item body does not contain %q: %s", want, itemBody)
				}
			}
			for _, unwanted := range []string{"First entry", "first@example.test"} {
				if strings.Contains(itemBody, unwanted) {
					t.Errorf("item body unexpectedly contains %q: %s", unwanted, itemBody)
				}
			}
		})
	}
}

func TestItemHandlerRendersOnlyTheEntryMatchingItsByteRange(t *testing.T) {
	source := []byte(`[
	{Name: "First entry", Email: "first@example.test"},
	{Name: "Second entry", Email: "second@example.test", Active: true},
]`)
	book, err := cuebook.New(source)
	if err != nil {
		t.Fatal(err)
	}

	var secondRange cuebook.ByteRange
	index := 0
	for entry, err := range book.EachEntry() {
		if err != nil {
			t.Fatal(err)
		}
		if index == 1 {
			secondRange, err = cuebook.NewByteRange(entry.Value)
			if err != nil {
				t.Fatal(err)
			}
			break
		}
		index++
	}

	const filePath = "people.cue"
	handler, err := New(fstest.MapFS{
		filePath: &fstest.MapFile{Data: source},
	})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		path       string
		head       string
		tail       string
		wantStatus int
		contains   []string
		omits      []string
	}{
		{
			name:       "renders matching entry as a single entry fragment",
			path:       filePath,
			head:       strconv.Itoa(secondRange.Head),
			tail:       strconv.Itoa(secondRange.Tail),
			wantStatus: http.StatusOK,
			contains:   []string{`data-entry-index="1"`, `data-file="people.cue"`, "Second entry", "second@example.test"},
			omits:      []string{"First entry", "first@example.test", "<!doctype html>"},
		},
		{
			name:       "missing file path is rejected",
			head:       strconv.Itoa(secondRange.Head),
			tail:       strconv.Itoa(secondRange.Tail),
			wantStatus: http.StatusBadRequest,
			contains:   []string{"The item request is invalid."},
		},
		{
			name:       "invalid range is rejected",
			path:       filePath,
			head:       "not-an-integer",
			tail:       strconv.Itoa(secondRange.Tail),
			wantStatus: http.StatusBadRequest,
			contains:   []string{"The item request is invalid."},
		},
		{
			name:       "unmatched range is not found",
			path:       filePath,
			head:       "0",
			tail:       "1",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "unknown file is not found",
			path:       "missing.cue",
			head:       strconv.Itoa(secondRange.Head),
			tail:       strconv.Itoa(secondRange.Tail),
			wantStatus: http.StatusNotFound,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			query := url.Values{}
			if test.path != "" {
				query.Set("path", test.path)
			}
			if test.head != "" {
				query.Set("head", test.head)
			}
			if test.tail != "" {
				query.Set("tail", test.tail)
			}

			request := httptest.NewRequest(http.MethodGet, "http://example.test/item?"+query.Encode(), nil)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", response.Code, test.wantStatus, response.Body.String())
			}

			body := response.Body.String()
			for _, want := range test.contains {
				if !strings.Contains(body, want) {
					t.Errorf("body does not contain %q: %s", want, body)
				}
			}
			for _, unwanted := range test.omits {
				if strings.Contains(body, unwanted) {
					t.Errorf("body unexpectedly contains %q: %s", unwanted, body)
				}
			}
			if test.wantStatus == http.StatusOK && strings.Count(body, `<article class="entry card"`) != 1 {
				t.Errorf("rendered entry count = %d, want 1: %s", strings.Count(body, `<article class="entry card"`), body)
			}
		})
	}
}
