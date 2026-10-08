package htmx

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/dkotik/cuebook"
)

func TestListAndEntryUseSeparateEntryTemplates(t *testing.T) {
	t.Parallel()

	const filePath = "people.cue"
	source := []byte(`[{Name: "Target entry"}]`)
	book, err := cuebook.New(source)
	if err != nil {
		t.Fatal(err)
	}
	var entryRange cuebook.ByteRange
	for entry, err := range book.EachEntry() {
		if err != nil {
			t.Fatal(err)
		}
		entryRange, err = cuebook.NewByteRange(entry.Value)
		if err != nil {
			t.Fatal(err)
		}
		break
	}

	handler, err := NewWithCommitter(fstest.MapFS{
		filePath: &fstest.MapFile{Data: source},
	}, &recordingCommitter{})
	if err != nil {
		t.Fatal(err)
	}

	listQuery := url.Values{"file": {filePath}}
	listRequest := httptest.NewRequest(http.MethodGet, "http://example.test/?"+listQuery.Encode(), nil)
	listResponse := httptest.NewRecorder()
	handler.ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d; body: %s", listResponse.Code, http.StatusOK, listResponse.Body.String())
	}
	if strings.Contains(listResponse.Body.String(), `action="/delete"`) {
		t.Errorf("list view unexpectedly includes the archive form: %s", listResponse.Body.String())
	}

	entryRequest := httptest.NewRequest(http.MethodGet, "http://example.test"+entryURL(filePath, entryRange), nil)
	entryResponse := httptest.NewRecorder()
	handler.ServeHTTP(entryResponse, entryRequest)
	if entryResponse.Code != http.StatusOK {
		t.Fatalf("entry status = %d, want %d; body: %s", entryResponse.Code, http.StatusOK, entryResponse.Body.String())
	}
	entryBody := entryResponse.Body.String()
	if !strings.Contains(entryBody, "<!doctype html>") {
		t.Errorf("normal entry view should render the full page: %s", entryBody)
	}
	if !strings.Contains(entryBody, `action="/delete"`) {
		t.Errorf("entry view does not include the archive form: %s", entryBody)
	}
}

func TestEntryHandlerRendersOnlyTheEntryMatchingItsByteRange(t *testing.T) {
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
			wantStatus: http.StatusInternalServerError,
			contains:   []string{"The entry request is invalid."},
		},
		{
			name:       "invalid range is rejected",
			path:       filePath,
			head:       "not-an-integer",
			tail:       strconv.Itoa(secondRange.Tail),
			wantStatus: http.StatusInternalServerError,
			contains:   []string{"The entry request is invalid."},
		},
		{
			name:       "unmatched range is not found",
			path:       filePath,
			head:       "0",
			tail:       "1",
			wantStatus: http.StatusNotFound,
			contains:   []string{"entry not found: path=people.cue byteRange=0-1"},
		},
		{
			name:       "unknown file is not found",
			path:       "missing.cue",
			head:       strconv.Itoa(secondRange.Head),
			tail:       strconv.Itoa(secondRange.Tail),
			wantStatus: http.StatusNotFound,
			contains:   []string{"file not found: path=missing.cue"},
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

			request := httptest.NewRequest(http.MethodGet, "http://example.test/entry?"+query.Encode(), nil)
			request.Header.Set("HX-Request", "true")
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
			if test.wantStatus == http.StatusOK && strings.Count(body, `<article class="card"`) != 1 {
				t.Errorf("rendered entry count = %d, want 1: %s", strings.Count(body, `<article class="card"`), body)
			}
		})
	}
}
