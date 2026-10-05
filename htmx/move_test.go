package htmx

import (
	"bytes"
	"errors"
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
	"github.com/dkotik/cuebook/patch"
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
	for _, want := range []string{`data-entry-drag-handle`, `draggable="true"`, `data-entry-index="0"`, `<a class="tree-file-link" data-file="contacts.cue"`} {
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

func TestEntryValueWithoutConstraintsRoundTripsPlainJSON(t *testing.T) {
	t.Parallel()

	source, err := fixtures.ReadFile("testdata/core1.cue")
	if err != nil {
		t.Fatal(err)
	}
	document, err := cuebook.New(source)
	if err != nil {
		t.Fatal(err)
	}
	entryValue, err := document.GetValue(0)
	if err != nil {
		t.Fatal(err)
	}

	decoded, intermediate, err := entryValueWithoutConstraints(entryValue)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(intermediate, "_#def") {
		t.Fatalf("intermediate entry encoding contains source definition references: %s", intermediate)
	}

	originalJSON, err := entryValue.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	decodedJSON, err := decoded.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decodedJSON, originalJSON) {
		t.Fatalf("round-tripped entry changed its concrete value: got %s, want %s", decodedJSON, originalJSON)
	}
}

func TestMoveTransfersEntriesBetweenFiles(t *testing.T) {
	t.Parallel()

	documents := transferTestDocuments()
	committer := &transferCommitter{files: transferSourceFS(documents)}
	handler, err := NewWithCommitter(committer.files, committer)
	if err != nil {
		t.Fatal(err)
	}

	response := submitTransfer(t, handler, "source.cue", 0, "destination.cue", true)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}
	body := response.Body.String()
	if !strings.Contains(body, `<main id="workspace"`) || !strings.Contains(body, `destination.cue`) {
		t.Fatalf("response does not display the destination workspace: %s", body)
	}
	existingPosition := strings.Index(body, `Existing destination`)
	movedPosition := strings.Index(body, `Move me`)
	if existingPosition < 0 || movedPosition < 0 || movedPosition < existingPosition {
		t.Fatalf("transferred entry was not appended after existing destination entries: %s", body)
	}

	if want := []string{"destination.cue", "source.cue"}; !equalStrings(committer.calls, want) {
		t.Fatalf("commit order = %v, want %v", committer.calls, want)
	}
	assertEntryTitles(t, committer.files["source.cue"].Data, []string{"Keep me", "Third source"})
	assertEntryTitles(t, committer.files["destination.cue"].Data, []string{"Existing destination", "Move me"})
}

func TestMoveTransferFailures(t *testing.T) {
	tests := []struct {
		name        string
		documents   map[string][]byte
		readOnly    bool
		destination string
		from        int
		failCalls   map[int]error
		wantStatus  int
		wantCalls   []string
		wantNotice  string
	}{
		{
			name:        "read-only source",
			documents:   transferTestDocuments(),
			readOnly:    true,
			destination: "destination.cue",
			wantStatus:  http.StatusForbidden,
			wantNotice:  "This source is read-only.",
		},
		{
			name:        "unknown destination",
			documents:   transferTestDocuments(),
			destination: "../outside.cue",
			wantStatus:  http.StatusNotFound,
			wantNotice:  "CUE file not found.",
		},
		{
			name: "destination constraint failure leaves both files unchanged",
			documents: map[string][]byte{
				"source.cue": []byte(`#entry: {Name: string @cuebook(title)}
[...#entry] & [{Name: "Move me"}]
`),
				"destination.cue": []byte(`#entry: {Name: "Allowed" @cuebook(title)}
[...#entry] & [{Name: "Allowed"}]
`),
			},
			destination: "destination.cue",
			wantStatus:  http.StatusUnprocessableEntity,
			wantNotice:  "does not satisfy the destination file",
		},
		{
			name: "source deletion constraint failure leaves both files unchanged",
			documents: map[string][]byte{
				"source.cue": []byte(`#entry: {Name: string @cuebook(title)}
[...#entry] & [{Name: "Move me"}, {Name: "Keep me"}] & [_, _, ...]
`),
				"destination.cue": []byte(`#entry: {Name: string @cuebook(title)}
[...#entry] & [{Name: "Existing destination"}]
`),
			},
			destination: "destination.cue",
			wantStatus:  http.StatusUnprocessableEntity,
			wantNotice:  "does not satisfy the source file",
		},
		{
			name:        "source commit failure rolls back destination",
			documents:   transferTestDocuments(),
			destination: "destination.cue",
			failCalls:   map[int]error{2: errors.New("source storage failure")},
			wantStatus:  http.StatusInternalServerError,
			wantCalls:   []string{"destination.cue", "source.cue", "destination.cue"},
			wantNotice:  "destination change was rolled back",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			files := transferSourceFS(test.documents)
			var handler http.Handler
			var committer *transferCommitter
			var err error
			if test.readOnly {
				handler, err = New(files)
			} else {
				committer = &transferCommitter{files: files, failCalls: test.failCalls}
				handler, err = NewWithCommitter(files, committer)
			}
			if err != nil {
				t.Fatal(err)
			}
			originalSource := append([]byte(nil), files["source.cue"].Data...)
			originalDestination := append([]byte(nil), files["destination.cue"].Data...)

			response := submitTransfer(t, handler, "source.cue", test.from, test.destination, true)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", response.Code, test.wantStatus, response.Body.String())
			}
			if !strings.Contains(response.Body.String(), test.wantNotice) {
				t.Errorf("response missing notice %q: %s", test.wantNotice, response.Body.String())
			}
			if committer != nil && !equalStrings(committer.calls, test.wantCalls) {
				t.Errorf("commit calls = %v, want %v", committer.calls, test.wantCalls)
			}
			if !bytes.Equal(files["source.cue"].Data, originalSource) {
				t.Errorf("source file changed after rejected transfer:\n%s", files["source.cue"].Data)
			}
			if !bytes.Equal(files["destination.cue"].Data, originalDestination) {
				t.Errorf("destination file changed after rejected transfer:\n%s", files["destination.cue"].Data)
			}
		})
	}
}

func transferTestDocuments() map[string][]byte {
	return map[string][]byte{
		"source.cue": []byte(`#entry: {Name: string @cuebook(title)}
[...#entry] & [{Name: "Move me"}, {Name: "Keep me"}, {Name: "Third source"}]
`),
		"destination.cue": []byte(`#entry: {Name: string @cuebook(title)}
[...#entry] & [{Name: "Existing destination"}]
`),
	}
}

func transferSourceFS(documents map[string][]byte) fstest.MapFS {
	files := make(fstest.MapFS, len(documents))
	for name, content := range documents {
		files[name] = &fstest.MapFile{Data: append([]byte(nil), content...)}
	}
	return files
}

type transferCommitter struct {
	files     fstest.MapFS
	calls     []string
	failCalls map[int]error
}

func (c *transferCommitter) Commit(name string, change patch.Patch) error {
	c.calls = append(c.calls, name)
	if err := c.failCalls[len(c.calls)]; err != nil {
		return err
	}
	file, ok := c.files[name]
	if !ok {
		return os.ErrNotExist
	}
	updated, err := change.ApplyToCueSource(file.Data)
	if err != nil {
		return err
	}
	if _, err := cuebook.New(updated); err != nil {
		return err
	}
	file.Data = updated
	return nil
}

func submitTransfer(t *testing.T, handler http.Handler, file string, from int, destination string, htmx bool) *httptest.ResponseRecorder {
	t.Helper()
	values := url.Values{
		"file":        {file},
		"from":        {strconv.Itoa(from)},
		"destination": {destination},
	}
	request := httptest.NewRequest(http.MethodPost, "http://example.test/move", strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Origin", "http://example.test")
	if htmx {
		request.Header.Set("HX-Request", "true")
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func assertEntryTitles(t *testing.T, source []byte, want []string) {
	t.Helper()
	titles, err := entryTitles(source)
	if err != nil {
		t.Fatalf("entryTitles(%q): %v", source, err)
	}
	if !equalStrings(titles, want) {
		t.Fatalf("entry titles = %v, want %v", titles, want)
	}
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range got {
		if got[index] != want[index] {
			return false
		}
	}
	return true
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
