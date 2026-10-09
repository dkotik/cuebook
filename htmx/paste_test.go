package htmx

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"testing/fstest"

	"cuelang.org/go/cue"
	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/patch"
)

func TestPasteEntry(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		initial   string
		source    string
		prefix    string
		wantTitle string
		wantCount int
	}{
		{name: "append struct", initial: `[{Name: "Existing"}]`, source: `{Name: "Pasted"}`, wantTitle: "Pasted", wantCount: 2},
		{name: "empty schema list", initial: "#entry: {Name: string}\n[...#entry] & []", source: `{Name: "First"}`, wantTitle: "First", wantCount: 1},
		{name: "unbraced struct", initial: `[]`, source: `Name: "Unbraced"`, wantTitle: "Unbraced", wantCount: 1},
		{name: "multiline source", initial: `[]`, source: "{\n// copied entry\nName: \"Café, & friends\"\nNotes: \"<script>\"\n}", wantTitle: "Café, & friends", wantCount: 1},
		{name: "mounted route", initial: `[]`, source: `{Name: "Mounted"}`, prefix: "/books", wantTitle: "Mounted", wantCount: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			files := fstest.MapFS{"contacts.cue": &fstest.MapFile{Data: []byte(tt.initial)}}
			handler, err := NewWithCommitter(files, mapFileCommitter{files: files}, WithServeMuxPrefix(tt.prefix))
			if err != nil {
				t.Fatal(err)
			}
			response := submitPaste(handler, tt.prefix+"/paste", url.Values{"file": {"contacts.cue"}, "source": {tt.source}})
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200; body: %s; flash: %s", response.Code, response.Body.String(), response.Header().Get("HX-Trigger"))
			}
			if !strings.Contains(response.Body.String(), `<main id="workspace"`) || strings.Contains(response.Body.String(), "<!doctype html>") {
				t.Fatalf("expected workspace fragment: %s", response.Body.String())
			}
			document, err := cuebook.New(files["contacts.cue"].Data)
			if err != nil {
				t.Fatal(err)
			}
			if count, err := document.Len(); err != nil || count != tt.wantCount {
				t.Fatalf("entry count = %d, err = %v; want %d", count, err, tt.wantCount)
			}
			value, err := document.GetValue(tt.wantCount - 1)
			if err != nil {
				t.Fatal(err)
			}
			if title, err := value.LookupPath(cue.ParsePath("Name")).String(); err != nil || title != tt.wantTitle {
				t.Errorf("saved title = %q, err = %v; want %q", title, err, tt.wantTitle)
			}
		})
	}
}

func TestPasteErrorsPreserveWorkspaceAndSource(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		source      string
		file        string
		readOnly    bool
		commitError error
		wantNotice  string
	}{
		{name: "empty clipboard", source: " \n\t", wantNotice: "pasted source is empty"},
		{name: "malformed CUE", source: `{Name:`, wantNotice: "Unable to parse"},
		{name: "plain text", source: "not a struct", wantNotice: "Unable to parse"},
		{name: "string", source: `"text"`, wantNotice: "must be a CUE struct"},
		{name: "list", source: `[{Name: "Pasted"}]`, wantNotice: "must be a CUE struct"},
		{name: "number", source: `42`, wantNotice: "must be a CUE struct"},
		{name: "constraint conflict", source: `{Name: 42}`, wantNotice: "does not satisfy the CUE constraints"},
		{name: "unknown field", source: `{Name: "Pasted", Extra: true}`, wantNotice: "does not satisfy the CUE constraints"},
		{name: "missing required field", source: `{}`, wantNotice: "does not satisfy the CUE constraints"},
		{name: "missing file", source: `{Name: "Pasted"}`, file: "missing.cue", wantNotice: "file not found"},
		{name: "invalid document", source: `{Name: "Pasted"}`, file: "invalid.cue", wantNotice: "Unable to parse or validate"},
		{name: "read only", source: `{Name: "Pasted"}`, readOnly: true, wantNotice: "read-only"},
		{name: "save failure", source: `{Name: "Pasted"}`, commitError: errors.New("disk full"), wantNotice: "could not be saved"},
		{name: "stale patch", source: `{Name: "Pasted"}`, commitError: patch.ErrByteRangeNotFound, wantNotice: "document changed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			original := []byte("#entry: {Name: string}\n[...#entry] & []")
			files := fstest.MapFS{"contacts.cue": &fstest.MapFile{Data: bytes.Clone(original)}, "invalid.cue": &fstest.MapFile{Data: []byte("[")}}
			committer := &recordingCommitter{err: tt.commitError}
			var handler http.Handler
			var err error
			if tt.readOnly {
				handler, err = New(files)
			} else {
				handler, err = NewWithCommitter(files, committer)
			}
			if err != nil {
				t.Fatal(err)
			}
			file := tt.file
			if file == "" {
				file = "contacts.cue"
			}
			response := submitPaste(handler, "/paste", url.Values{"file": {file}, "source": {tt.source}})
			if response.Code != http.StatusNoContent || response.Body.Len() != 0 {
				t.Fatalf("status = %d, body = %q; want empty 204", response.Code, response.Body.String())
			}
			var trigger map[string]struct {
				Message string `json:"message"`
			}
			if err := json.Unmarshal([]byte(response.Header().Get("HX-Trigger")), &trigger); err != nil {
				t.Fatal(err)
			}
			if message := trigger["cuebook:flash"].Message; !strings.Contains(message, tt.wantNotice) {
				t.Errorf("flash = %q, want %q", message, tt.wantNotice)
			}
			if !bytes.Equal(files["contacts.cue"].Data, original) {
				t.Fatal("rejected paste changed source")
			}
			wantCalls := 0
			if tt.commitError != nil {
				wantCalls = 1
			}
			if committer.calls != wantCalls {
				t.Errorf("commit calls = %d, want %d", committer.calls, wantCalls)
			}
		})
	}
}

func submitPaste(handler http.Handler, route string, values url.Values) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "http://example.test"+route, strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("HX-Request", "true")
	request.Header.Set("Origin", "http://example.test")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
