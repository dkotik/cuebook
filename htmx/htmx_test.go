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

func TestAddEntryFormUsesSchemaFields(t *testing.T) {
	t.Parallel()

	handler, err := NewWithCommitter(testSource(), &recordingCommitter{})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "http://example.test/?file=core1.cue", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}

	for _, want := range []string{
		`action="/add"`,
		`hx-post="/add"`,
		`name="field" value="Name"`,
		`name="field" value="Email"`,
		`name="field" value="Notes"`,
		`name="field" value="Password"`,
		"(optional)",
		"Add entry",
	} {
		if !strings.Contains(response.Body.String(), want) {
			t.Errorf("page does not contain %q", want)
		}
	}
}

func TestAddEntryFormStartsAtTypeZeroValues(t *testing.T) {
	t.Parallel()

	source := []byte(`#entry: {
	Name: *"Default name" | string
	Count: *42 | int
	Enabled: *true | bool
}
[...#entry] & [{Name: "Existing", Count: 7, Enabled: true}]
`)
	handler, err := NewWithCommitter(fstest.MapFS{
		"defaults.cue": &fstest.MapFile{Data: source},
	}, &recordingCommitter{})
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "http://example.test/?file=defaults.cue", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}
	body := response.Body.String()
	start, end := strings.Index(body, `<section class="add-entry`), strings.Index(body, `</section>`)
	if start < 0 || end < start {
		t.Fatalf("add-entry form not found: %s", body)
	}
	form := body[start:end]
	for _, want := range []string{
		`name="field" value="Name"`,
		`name="value" value="" required`,
		`name="field" value="Count"`,
		`type="number" name="value" value="0" step="any" required`,
		`name="field" value="Enabled"`,
		`<option value="false" selected>false</option>`,
		`<option value="true">true</option>`,
	} {
		if !strings.Contains(form, want) {
			t.Errorf("add-entry form does not contain %q: %s", want, form)
		}
	}
	for _, unwanted := range []string{"Default name", `value="42"`, `<option value="true" selected>`} {
		if strings.Contains(form, unwanted) {
			t.Errorf("add-entry form unexpectedly contains %q", unwanted)
		}
	}
}

func TestEditableFieldsUseInlineHTMXEditor(t *testing.T) {
	t.Parallel()

	handler, err := NewWithCommitter(testSource(), &recordingCommitter{})
	if err != nil {
		t.Fatal(err)
	}

	pageRequest := httptest.NewRequest(http.MethodGet, "http://example.test/?file=core1.cue", nil)
	pageResponse := httptest.NewRecorder()
	handler.ServeHTTP(pageResponse, pageRequest)
	if pageResponse.Code != http.StatusOK {
		t.Fatalf("page status = %d, want %d; body: %s", pageResponse.Code, http.StatusOK, pageResponse.Body.String())
	}
	for _, want := range []string{
		`<output>First11111aa1</output>`,
		`field-edit-button`,
		`aria-label="Edit Name"`,
		`hx-get="/edit?entry=0&amp;field=Name&amp;file=core1.cue"`,
	} {
		if !strings.Contains(pageResponse.Body.String(), want) {
			t.Errorf("writable page does not contain %q; body: %s", want, pageResponse.Body.String())
		}
	}
	if strings.Contains(pageResponse.Body.String(), `hx-post="/edit"`) {
		t.Fatal("field edit forms should only be rendered after clicking the pencil")
	}

	formQuery := url.Values{"entry": {"0"}, "field": {"Name"}, "file": {"core1.cue"}}
	formRequest := httptest.NewRequest(http.MethodGet, "http://example.test/edit?"+formQuery.Encode(), nil)
	formRequest.Header.Set("HX-Request", "true")
	formResponse := httptest.NewRecorder()
	handler.ServeHTTP(formResponse, formRequest)
	if formResponse.Code != http.StatusOK {
		t.Fatalf("form status = %d, want %d; body: %s", formResponse.Code, http.StatusOK, formResponse.Body.String())
	}
	for _, want := range []string{
		`<form action="/edit" method="post"`,
		`hx-post="/edit"`,
		`hx-target="#workspace"`,
		`name="value" value="First11111aa1"`,
		`hx-get="/edit?entry=0&amp;field=Name&amp;file=core1.cue&amp;mode=view"`,
		"Cancel",
	} {
		if !strings.Contains(formResponse.Body.String(), want) {
			t.Errorf("edit form does not contain %q", want)
		}
	}
	if strings.Contains(formResponse.Body.String(), "<!doctype html>") || strings.Contains(formResponse.Body.String(), `<main id="workspace"`) {
		t.Fatal("expected a field-level fragment, not a full page")
	}

	formQuery.Set("mode", "view")
	viewRequest := httptest.NewRequest(http.MethodGet, "http://example.test/edit?"+formQuery.Encode(), nil)
	viewResponse := httptest.NewRecorder()
	handler.ServeHTTP(viewResponse, viewRequest)
	if viewResponse.Code != http.StatusOK {
		t.Fatalf("view status = %d, want %d; body: %s", viewResponse.Code, http.StatusOK, viewResponse.Body.String())
	}
	if !strings.Contains(viewResponse.Body.String(), `<output>First11111aa1</output>`) || !strings.Contains(viewResponse.Body.String(), `field-edit-button`) {
		t.Fatalf("cancel response did not restore the static field view: %s", viewResponse.Body.String())
	}
	if strings.Contains(viewResponse.Body.String(), `<form action="/edit"`) {
		t.Fatal("cancel response unexpectedly contains an edit form")
	}
}

func TestReadOnlySourceRejectsOpeningEditForm(t *testing.T) {
	handler, err := New(testSource())
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "http://example.test/edit?entry=0&field=Name&file=core1.cue", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusForbidden, response.Body.String())
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
		contains   []string
		wantStatus int
	}{
		{name: "version-pinned htmx", path: "/assets/htmx-2.0.4.min.js", contains: []string{"htmx"}, wantStatus: http.StatusOK},
		{name: "version-pinned Bulma", path: "/assets/bulma.css", contains: []string{"bulma.io v1.0.4"}, wantStatus: http.StatusOK},
		{name: "Bulma license", path: "/assets/bulma-LICENSE.txt", contains: []string{"The MIT License"}, wantStatus: http.StatusOK},
		{name: "SVG book favicon", path: "/assets/favicon.svg", contains: []string{"<svg", `fill="#22c55e"`, `fill="#3b82f6"`}, wantStatus: http.StatusOK},
		{name: "htmx license", path: "/assets/htmx-LICENSE.txt", contains: []string{"Zero-Clause BSD"}, wantStatus: http.StatusOK},
		{name: "stylesheet", path: "/assets/app.css", contains: []string{"grid-template-columns", "file-tree-node", ".tree-chevron", ".tree-children[hidden]"}, wantStatus: http.StatusOK},
		{name: "file tree component", path: "/assets/file-tree.js", contains: []string{"cuebook-file-tree-folded", `customElements.define("file-tree-node"`, "readFoldedPaths", "writeFoldedPaths", `setAttribute("aria-expanded"`, "syncCurrentFile", `htmx:pushedIntoHistory`}, wantStatus: http.StatusOK},
		{name: "theme controller", path: "/assets/theme.js", contains: []string{"localStorage.setItem", `querySelectorAll("[data-theme-icon]")`, `? "moon" : "sun"`, `icon.style.display`, `? "inline-block" : "none"`}, wantStatus: http.StatusOK},
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
			for _, content := range tt.contains {
				if !strings.Contains(response.Body.String(), content) {
					t.Errorf("asset body does not contain %q", content)
				}
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

func TestWritableDirectoryAppendsEntries(t *testing.T) {
	tests := []struct {
		name string
		htmx bool
	}{
		{name: "browser form redirects after append"},
		{name: "htmx form receives updated fragment", htmx: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			directory := t.TempDir()
			source, err := fixtures.ReadFile("testdata/core1.cue")
			if err != nil {
				t.Fatal(err)
			}
			before, err := cuebook.New(source)
			if err != nil {
				t.Fatal(err)
			}
			beforeLength, err := before.Len()
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

			response := submitAdd(t, handler, tt.htmx, addEntryValues("contacts.cue", [][2]string{
				{"Name", "Added Contact"},
				{"Email", "added@example.test"},
				{"Notes", ""},
				{"Password", ""},
			}))
			if tt.htmx {
				if response.Code != http.StatusOK {
					t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
				}
				if !strings.Contains(response.Body.String(), `<main id="workspace"`) || !strings.Contains(response.Body.String(), "Added Contact") {
					t.Fatalf("expected updated HTMX workspace, got: %s", response.Body.String())
				}
			} else {
				if response.Code != http.StatusSeeOther {
					t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusSeeOther, response.Body.String())
				}
				if !strings.Contains(response.Header().Get("Location"), "contacts.cue") {
					t.Errorf("redirect location = %q", response.Header().Get("Location"))
				}
			}

			updated, err := os.ReadFile(filepath.Join(directory, "contacts.cue"))
			if err != nil {
				t.Fatal(err)
			}
			document, err := cuebook.New(updated)
			if err != nil {
				t.Fatalf("saved source is invalid: %v", err)
			}
			length, err := document.Len()
			if err != nil || length != beforeLength+1 {
				t.Fatalf("entry count = %d, err = %v; want %d", length, err, beforeLength+1)
			}
			value, err := document.GetValue(length - 1)
			if err != nil {
				t.Fatal(err)
			}
			entry, err := cuebook.NewEntry(value)
			if err != nil {
				t.Fatal(err)
			}
			for name, want := range map[string]string{"Name": "Added Contact", "Email": "added@example.test"} {
				field, ok := entry.GetFieldByName(name)
				if !ok || field.String() != want {
					t.Errorf("saved %s = %q, found = %t; want %q", name, field.String(), ok, want)
				}
			}
		})
	}
}

func TestWritableDirectoryAppendsToEmptySchemaList(t *testing.T) {
	directory := t.TempDir()
	source := []byte(`#email: =~"^[^@]+@[^@]+$"
#contact: {
	Name: string @cuebook(title)
	Email: #email
}
[...#contact] & []
`)
	if err := os.WriteFile(filepath.Join(directory, "contacts.cue"), source, 0o600); err != nil {
		t.Fatal(err)
	}
	handler, err := NewDirectory(directory)
	if err != nil {
		t.Fatal(err)
	}

	response := submitAdd(t, handler, false, addEntryValues("contacts.cue", [][2]string{
		{"Name", "First Contact"},
		{"Email", "first@example.test"},
	}))
	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusSeeOther, response.Body.String())
	}

	updated, err := os.ReadFile(filepath.Join(directory, "contacts.cue"))
	if err != nil {
		t.Fatal(err)
	}
	document, err := cuebook.New(updated)
	if err != nil {
		t.Fatalf("saved source is invalid: %v", err)
	}
	if length, err := document.Len(); err != nil || length != 1 {
		t.Fatalf("entry count = %d, err = %v; want 1", length, err)
	}
	value, err := document.GetValue(0)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := cuebook.NewEntry(value)
	if err != nil {
		t.Fatal(err)
	}
	if field, ok := entry.GetFieldByName("Name"); !ok || field.String() != "First Contact" {
		t.Fatalf("saved Name = %q, found = %t", field.String(), ok)
	}
}

func TestAddEntryFailures(t *testing.T) {
	committer := &recordingCommitter{}
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
			name: "invalid email is rejected by CUE",
			values: addEntryValues("core1.cue", [][2]string{
				{"Name", "Invalid Contact"}, {"Email", "not-an-email"},
			}),
			origin:     "http://example.test",
			wantStatus: http.StatusUnprocessableEntity,
			wantNotice: "does not satisfy the CUE constraints",
		},
		{
			name: "unknown field is rejected",
			values: addEntryValues("core1.cue", [][2]string{
				{"Name", "Contact"}, {"Email", "valid@example.test"}, {"notAField", "value"},
			}),
			origin:     "http://example.test",
			wantStatus: http.StatusBadRequest,
			wantNotice: "unknown field",
		},
		{
			name:       "missing required field is rejected",
			values:     addEntryValues("core1.cue", [][2]string{{"Name", "Contact"}}),
			origin:     "http://example.test",
			wantStatus: http.StatusBadRequest,
			wantNotice: "required entry field is missing",
		},
		{
			name: "duplicate field is rejected",
			values: addEntryValues("core1.cue", [][2]string{
				{"Name", "Contact"}, {"Name", "Other Contact"}, {"Email", "valid@example.test"},
			}),
			origin:     "http://example.test",
			wantStatus: http.StatusBadRequest,
			wantNotice: "duplicate field",
		},
		{
			name: "cross-origin form is rejected",
			values: addEntryValues("core1.cue", [][2]string{
				{"Name", "Contact"}, {"Email", "valid@example.test"},
			}),
			origin:     "https://attacker.test",
			wantStatus: http.StatusForbidden,
			wantNotice: "Cross-origin edits are not allowed.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := submitAddWithOrigin(t, handler, false, tt.values, tt.origin)
			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", response.Code, tt.wantStatus, response.Body.String())
			}
			if !strings.Contains(response.Body.String(), tt.wantNotice) {
				t.Errorf("body does not contain %q", tt.wantNotice)
			}
		})
	}
	if committer.calls != 0 {
		t.Fatalf("committer calls = %d for rejected additions, want 0", committer.calls)
	}
}

func TestReadOnlySourceRejectsAdding(t *testing.T) {
	handler, err := New(testSource())
	if err != nil {
		t.Fatal(err)
	}
	response := submitAdd(t, handler, false, addEntryValues("core1.cue", [][2]string{
		{"Name", "Contact"}, {"Email", "valid@example.test"},
	}))
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusForbidden, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "This source is read-only.") {
		t.Fatal("expected read-only explanation")
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

func submitAdd(t *testing.T, handler http.Handler, htmx bool, values url.Values) *httptest.ResponseRecorder {
	t.Helper()
	return submitAddWithOrigin(t, handler, htmx, values, "http://example.test")
}

func submitAddWithOrigin(t *testing.T, handler http.Handler, htmx bool, values url.Values, origin string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "http://example.test/add", strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Origin", origin)
	if htmx {
		request.Header.Set("HX-Request", "true")
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func addEntryValues(fileName string, fields [][2]string) url.Values {
	values := url.Values{"file": {fileName}}
	for _, field := range fields {
		values.Add("field", field[0])
		values.Add("value", field[1])
	}
	return values
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
