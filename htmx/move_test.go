package htmx

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/dkotik/cuebook"
)

func TestWritableDirectoryMovesEntries(t *testing.T) {
	t.Parallel()

	source, err := fixtures.ReadFile("testdata/core1.cue")
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	filePath := filepath.Join(directory, "contacts.cue")
	if err := os.WriteFile(filePath, source, 0o600); err != nil {
		t.Fatal(err)
	}
	handler, err := NewDirectory(directory)
	if err != nil {
		t.Fatal(err)
	}

	pageRequest := httptest.NewRequest(http.MethodGet, "http://example.test/?file=contacts.cue", nil)
	pageResponse := httptest.NewRecorder()
	handler.ServeHTTP(pageResponse, pageRequest)
	if pageResponse.Code != http.StatusOK {
		t.Fatalf("page status = %d, want %d; body: %s", pageResponse.Code, http.StatusOK, pageResponse.Body.String())
	}
	for _, want := range []string{`data-entry-drag-handle`, `draggable="true"`, `data-entry-index="0"`, `data-file="contacts.cue"`} {
		if !strings.Contains(pageResponse.Body.String(), want) {
			t.Errorf("writable page does not contain %q", want)
		}
	}

	response := submitMove(t, handler, "contacts.cue", 2, 0, "http://example.test", true)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `<main id="workspace"`) {
		t.Fatalf("expected an updated HTMX workspace response: %s", response.Body.String())
	}
	newEntryPosition := strings.Index(response.Body.String(), `<h2 class="card-header-title">new entry</h2>`)
	firstEntryPosition := strings.Index(response.Body.String(), `<h2 class="card-header-title">First11111aa1</h2>`)
	if newEntryPosition < 0 || firstEntryPosition < newEntryPosition {
		t.Fatalf("response did not reflect the new entry order: %s", response.Body.String())
	}

	updated, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	titles, err := entryTitles(updated)
	if err != nil {
		t.Fatal(err)
	}
	wantTitles := []string{"new entry", "First11111aa1", "First11111aa1", "First11111aa1axx"}
	if len(titles) != len(wantTitles) {
		t.Fatalf("entry count = %d, want %d: %v", len(titles), len(wantTitles), titles)
	}
	for index := range wantTitles {
		if titles[index] != wantTitles[index] {
			t.Errorf("entry %d title = %q, want %q; all titles: %v", index, titles[index], wantTitles[index], titles)
		}
	}
}

func TestMoveRejectsReadOnlyAndInvalidIndexes(t *testing.T) {
	t.Parallel()

	readOnly, err := New(testSource())
	if err != nil {
		t.Fatal(err)
	}
	response := submitMove(t, readOnly, "core1.cue", 0, 1, "http://example.test", true)
	if response.Code != http.StatusForbidden {
		t.Fatalf("read-only status = %d, want %d; body: %s", response.Code, http.StatusForbidden, response.Body.String())
	}

	committer := &recordingCommitter{}
	writable, err := NewWithCommitter(testSource(), committer)
	if err != nil {
		t.Fatal(err)
	}
	response = submitMove(t, writable, "core1.cue", 0, 100, "http://example.test", true)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("invalid position status = %d, want %d; body: %s", response.Code, http.StatusBadRequest, response.Body.String())
	}
	if committer.calls != 0 {
		t.Fatalf("committer calls = %d, want 0", committer.calls)
	}
}

func TestMoveRejectsInvalidReorderedDocument(t *testing.T) {
	t.Parallel()

	source := []byte(`[ {Name: "one"}, {Name: "two"} ] & [ {Name: "one"}, {Name: "two"} ]`)
	committer := &recordingCommitter{}
	handler, err := NewWithCommitter(fstest.MapFS{
		"ordered.cue": &fstest.MapFile{Data: source},
	}, committer)
	if err != nil {
		t.Fatal(err)
	}
	response := submitMove(t, handler, "ordered.cue", 0, 1, "http://example.test", false)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusUnprocessableEntity, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "does not satisfy the CUE constraints") {
		t.Fatalf("validation error missing from response: %s", response.Body.String())
	}
	if committer.calls != 0 {
		t.Fatalf("committer calls = %d for invalid reorder, want 0", committer.calls)
	}
}

func TestEntryMovePatchInverts(t *testing.T) {
	t.Parallel()

	source, err := fixtures.ReadFile("testdata/core1.cue")
	if err != nil {
		t.Fatal(err)
	}
	change, err := entryMovePatch(source, 2, 0)
	if err != nil {
		t.Fatal(err)
	}
	moved, err := change.ApplyToCueSource(source)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := change.Invert().ApplyToCueSource(moved)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(restored, source) {
		t.Fatalf("inverse move patch did not restore original source:\n%s", restored)
	}
}

func submitMove(t *testing.T, handler http.Handler, file string, from, to int, origin string, htmx bool) *httptest.ResponseRecorder {
	t.Helper()
	values := url.Values{
		"file": {file},
		"from": {strconv.Itoa(from)},
		"to":   {strconv.Itoa(to)},
	}
	request := httptest.NewRequest(http.MethodPost, "http://example.test/move", strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Origin", origin)
	if htmx {
		request.Header.Set("HX-Request", "true")
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func entryTitles(source []byte) ([]string, error) {
	document, err := cuebook.New(source)
	if err != nil {
		return nil, err
	}
	length, err := document.Len()
	if err != nil {
		return nil, err
	}
	titles := make([]string, 0, length)
	for index := 0; index < length; index++ {
		value, err := document.GetValue(index)
		if err != nil {
			return nil, err
		}
		entry, err := cuebook.NewEntry(value)
		if err != nil {
			return nil, err
		}
		titles = append(titles, entry.GetTitle())
	}
	return titles, nil
}
