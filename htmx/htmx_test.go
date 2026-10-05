package htmx

import (
	"embed"
	"errors"

	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/patch"
)

//go:embed testdata/core1.cue testdata/subfolder/sub1.cue
var fixtures embed.FS

func testSource() fstest.MapFS {
	root, err := fixtures.ReadFile("testdata/core1.cue")
	if err != nil {
		panic(err)
	}
	nested, err := fixtures.ReadFile("testdata/subfolder/sub1.cue")
	if err != nil {
		panic(err)
	}
	return fstest.MapFS{
		"core1.cue":          &fstest.MapFile{Data: root},
		"subfolder/sub1.cue": &fstest.MapFile{Data: nested},
		"notes.txt":          &fstest.MapFile{Data: []byte("not a CUE file")},
		"invalid.cue":        &fstest.MapFile{Data: []byte("[")},
		"escaped.cue":        &fstest.MapFile{Data: []byte(`[ { Name: "<script>alert(1)</script>" } ]`)},
	}
}

func TestReadOnlyHandler(t *testing.T) {
	handler, err := New(testSource())
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		method     string
		path       string
		htmx       bool
		wantStatus int
		contains   []string
		omits      []string
	}{
		{
			name:       "index defaults to dark and offers a theme toggle",
			method:     http.MethodGet,
			path:       "/",
			wantStatus: http.StatusOK,
			contains:   []string{"core1.cue", "subfolder/sub1.cue", "file=subfolder%2Fsub1.cue", "bulma.css", "htmx-2.0.4.min.js", "/assets/theme.js", `data-theme="dark"`, `id="theme-toggle"`, `aria-pressed="true"`, "Dark mode"},
			omits:      []string{"notes.txt", "First11111aa"},
		},
		{
			name:       "selected nested file renders entries",
			method:     http.MethodGet,
			path:       "/?file=subfolder%2Fsub1.cue",
			wantStatus: http.StatusOK,
			contains:   []string{"subfolder/sub1.cue", "First11111aa", "test1@testdomain.com", "Read-only source"},
			omits:      []string{`hx-post="/edit"`},
		},
		{
			name:       "htmx request gets workspace fragment",
			method:     http.MethodGet,
			path:       "/?file=core1.cue",
			htmx:       true,
			wantStatus: http.StatusOK,
			contains:   []string{`<main id="workspace"`, "First11111aa"},
			omits:      []string{"<!doctype html>", "file-nav"},
		},
		{
			name:       "path traversal is not found",
			method:     http.MethodGet,
			path:       "/?file=..%2Fsecret.cue",
			wantStatus: http.StatusNotFound,
			contains:   []string{"CUE file not found."},
		},
		{
			name:       "missing file is not found",
			method:     http.MethodGet,
			path:       "/?file=missing.cue",
			wantStatus: http.StatusNotFound,
			contains:   []string{"CUE file not found."},
		},
		{
			name:       "invalid cue is reported safely",
			method:     http.MethodGet,
			path:       "/?file=invalid.cue",
			wantStatus: http.StatusUnprocessableEntity,
			contains:   []string{"Unable to parse or validate this CUE document"},
		},
		{
			name:       "untrusted cue content is escaped",
			method:     http.MethodGet,
			path:       "/?file=escaped.cue",
			wantStatus: http.StatusOK,
			contains:   []string{`&lt;script&gt;alert(1)&lt;/script&gt;`},
			omits:      []string{`<script>alert(1)</script>`},
		},
		{
			name:       "unsupported method is rejected",
			method:     http.MethodPut,
			path:       "/",
			wantStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(tt.method, "http://example.test"+tt.path, nil)
			if tt.htmx {
				request.Header.Set("HX-Request", "true")
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", response.Code, tt.wantStatus, response.Body.String())
			}
			body := response.Body.String()
			for _, want := range tt.contains {
				if !strings.Contains(body, want) {
					t.Errorf("body does not contain %q", want)
				}
			}
			for _, unwanted := range tt.omits {
				if strings.Contains(body, unwanted) {
					t.Errorf("body unexpectedly contains %q", unwanted)
				}
			}
		})
	}
}

func TestAssetsAreServedLocally(t *testing.T) {
	handler, err := New(testSource())
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		path       string
		content    string
		wantStatus int
	}{
		{name: "version-pinned htmx", path: "/assets/htmx-2.0.4.min.js", content: "htmx", wantStatus: http.StatusOK},
		{name: "version-pinned Bulma", path: "/assets/bulma.css", content: "bulma.io v1.0.4", wantStatus: http.StatusOK},
		{name: "Bulma license", path: "/assets/bulma-LICENSE.txt", content: "The MIT License", wantStatus: http.StatusOK},
		{name: "htmx license", path: "/assets/htmx-LICENSE.txt", content: "Zero-Clause BSD", wantStatus: http.StatusOK},
		{name: "stylesheet", path: "/assets/app.css", content: "grid-template-columns", wantStatus: http.StatusOK},
		{name: "theme controller", path: "/assets/theme.js", content: "localStorage.setItem", wantStatus: http.StatusOK},
		{name: "unknown asset", path: "/assets/secret.txt", wantStatus: http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "http://example.test"+tt.path, nil)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, tt.wantStatus)
			}
			if tt.content != "" && !strings.Contains(response.Body.String(), tt.content) {
				t.Errorf("asset body does not contain %q", tt.content)
			}
		})
	}
}

func TestWritableDirectoryCommitsEdits(t *testing.T) {
	tests := []struct {
		name       string
		htmx       bool
		wantStatus int
		fragment   bool
	}{
		{name: "browser form redirects after save", wantStatus: http.StatusSeeOther},
		{name: "htmx form receives updated fragment", htmx: true, wantStatus: http.StatusOK, fragment: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			directory := t.TempDir()
			source, err := fixtures.ReadFile("testdata/core1.cue")
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(directory, "contacts.cue"), source, 0o600); err != nil {
				t.Fatal(err)
			}
			handler, err := NewDirectory(directory)
			if err != nil {
				t.Fatal(err)
			}

			response := submitEdit(t, handler, tt.htmx, url.Values{
				"file":  {"contacts.cue"},
				"entry": {"0"},
				"field": {"Name"},
				"value": {"Saved Contact"},
			})
			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", response.Code, tt.wantStatus, response.Body.String())
			}
			if tt.fragment {
				if !strings.Contains(response.Body.String(), `<main id="workspace"`) || strings.Contains(response.Body.String(), "<!doctype html>") {
					t.Fatalf("expected HTMX workspace fragment, got: %s", response.Body.String())
				}
				if !strings.Contains(response.Body.String(), "Saved Contact") {
					t.Errorf("updated value missing from response: %s", response.Body.String())
				}
			}
			if tt.wantStatus == http.StatusSeeOther && !strings.Contains(response.Header().Get("Location"), "contacts.cue") {
				t.Errorf("redirect location = %q", response.Header().Get("Location"))
			}

			updated, err := os.ReadFile(filepath.Join(directory, "contacts.cue"))
			if err != nil {
				t.Fatal(err)
			}
			document, err := cuebook.New(updated)
			if err != nil {
				t.Fatalf("saved source is invalid: %v", err)
			}
			value, err := document.GetValue(0)
			if err != nil {
				t.Fatal(err)
			}
			entry, err := cuebook.NewEntry(value)
			if err != nil {
				t.Fatal(err)
			}
			field, ok := entry.GetFieldByName("Name")
			if !ok || field.String() != "Saved Contact" {
				t.Fatalf("saved Name = %q, found = %t", field.String(), ok)
			}
		})
	}
}

func TestEditFailures(t *testing.T) {
	committer := &recordingCommitter{err: errors.New("private disk error")}
	handler, err := NewWithCommitter(testSource(), committer)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		values     url.Values
		origin     string
		wantStatus int
		wantNotice string
	}{
		{
			name:       "invalid email is rejected by cue validation",
			values:     editValues("core1.cue", "0", "Email", "not-an-email"),
			origin:     "http://example.test",
			wantStatus: http.StatusUnprocessableEntity,
			wantNotice: "does not satisfy the CUE constraints",
		},
		{
			name:       "missing field is not found",
			values:     editValues("core1.cue", "0", "notAField", "value"),
			origin:     "http://example.test",
			wantStatus: http.StatusNotFound,
			wantNotice: "Field not found.",
		},
		{
			name:       "out of range entry is not found",
			values:     editValues("core1.cue", "9", "Name", "value"),
			origin:     "http://example.test",
			wantStatus: http.StatusNotFound,
			wantNotice: "Entry not found.",
		},
		{
			name:       "path traversal is not found",
			values:     editValues("../core1.cue", "0", "Name", "value"),
			origin:     "http://example.test",
			wantStatus: http.StatusNotFound,
			wantNotice: "CUE file not found.",
		},
		{
			name:       "cross-origin submit is forbidden",
			values:     editValues("core1.cue", "0", "Name", "value"),
			origin:     "https://attacker.test",
			wantStatus: http.StatusForbidden,
			wantNotice: "Cross-origin edits are not allowed.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := submitEditWithOrigin(t, handler, false, tt.values, tt.origin)
			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", response.Code, tt.wantStatus, response.Body.String())
			}
			if !strings.Contains(response.Body.String(), tt.wantNotice) {
				t.Errorf("body does not contain %q", tt.wantNotice)
			}
		})
	}
	if committer.calls != 0 {
		t.Errorf("committer called %d times for rejected edits, want 0", committer.calls)
	}
}

func TestReadOnlySourceRejectsEdits(t *testing.T) {
	handler, err := New(testSource())
	if err != nil {
		t.Fatal(err)
	}
	response := submitEdit(t, handler, false, editValues("core1.cue", "0", "Name", "Updated"))
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusForbidden, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "This source is read-only.") {
		t.Fatal("expected read-only explanation")
	}
}

func TestCommitFailureDoesNotExposeStorageError(t *testing.T) {
	committer := &recordingCommitter{err: errors.New("private disk error")}
	handler, err := NewWithCommitter(testSource(), committer)
	if err != nil {
		t.Fatal(err)
	}
	response := submitEdit(t, handler, false, editValues("core1.cue", "0", "Name", "Updated"))
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusInternalServerError, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "private disk error") {
		t.Fatal("storage error leaked to response")
	}
	if !strings.Contains(response.Body.String(), "The edit could not be saved.") {
		t.Fatal("expected safe storage error message")
	}
	if committer.calls != 1 {
		t.Fatalf("committer calls = %d, want 1", committer.calls)
	}
}

func TestConstructorsValidateDependencies(t *testing.T) {
	tests := []struct {
		name string
		call func() (http.Handler, error)
	}{
		{name: "nil source", call: func() (http.Handler, error) { return New(nil) }},
		{name: "nil committer", call: func() (http.Handler, error) { return NewWithCommitter(testSource(), nil) }},
		{name: "missing directory", call: func() (http.Handler, error) { return NewDirectory(filepath.Join(t.TempDir(), "missing")) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, err := tt.call()
			if err == nil {
				t.Fatal("expected an error")
			}
			if handler != nil {
				t.Fatal("handler should be nil on constructor error")
			}
		})
	}
}

func submitEdit(t *testing.T, handler http.Handler, htmx bool, values url.Values) *httptest.ResponseRecorder {
	t.Helper()
	return submitEditWithOrigin(t, handler, htmx, values, "http://example.test")
}

func submitEditWithOrigin(t *testing.T, handler http.Handler, htmx bool, values url.Values, origin string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "http://example.test/edit", strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Origin", origin)
	if htmx {
		request.Header.Set("HX-Request", "true")
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func editValues(fileName, index, field, value string) url.Values {
	return url.Values{
		"file":  {fileName},
		"entry": {index},
		"field": {field},
		"value": {value},
	}
}

type recordingCommitter struct {
	calls int
	err   error
}

func (c *recordingCommitter) Commit(_ string, _ patch.Patch) error {
	c.calls++
	return c.err
}
