package htmx

import (
	"errors"
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/dkotik/cuebook"
)

func TestEditFormRendersFieldDocumentationBelowInput(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		wantHelp    string
		wantHelpNum int
	}{
		{
			name: "escaped documentation appears after the input",
			source: `[
				{
					Name: "Ada",
					// Use a monitored <work> address & update it when needed.
					Email: "ada@example.test",
				}
			]`,
			wantHelp:    `<p class="help">Use a monitored &lt;work&gt; address &amp; update it when needed.</p>`,
			wantHelpNum: 1,
		},
		{
			name:        "field without documentation has no help paragraph",
			source:      `[{Name: "Ada", Email: "ada@example.test"}]`,
			wantHelpNum: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			handler, err := NewWithCommitter(fstest.MapFS{
				"people.cue": &fstest.MapFile{Data: []byte(test.source)},
			}, &recordingCommitter{})
			if err != nil {
				t.Fatal(err)
			}

			query := url.Values{"entry": {"0"}, "field": {"Email"}, "file": {"people.cue"}}
			request := httptest.NewRequest(http.MethodGet, "http://example.test/edit?"+query.Encode(), nil)
			request.Header.Set("HX-Request", "true")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
			}

			body := response.Body.String()
			if got := strings.Count(body, `<p class="help">`); got != test.wantHelpNum {
				t.Fatalf("help paragraph count = %d, want %d: %s", got, test.wantHelpNum, body)
			}
			if test.wantHelp != "" {
				inputPosition := strings.Index(body, `value="ada@example.test"`)
				helpPosition := strings.Index(body, test.wantHelp)
				if inputPosition < 0 || helpPosition < inputPosition {
					t.Fatalf("description should appear after the input: %s", body)
				}
			}
		})
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
		`<div class="field field-item" title="Email">`,
		`<output>`,
		`field-value-edit`,
		`aria-label="Edit Email value"`,
		`hx-get="/edit?entry=0&amp;field=Email&amp;file=core1.cue"`,
	} {
		if !strings.Contains(pageResponse.Body.String(), want) {
			t.Errorf("writable page does not contain %q; body: %s", want, pageResponse.Body.String())
		}
	}
	if strings.Contains(pageResponse.Body.String(), `field-edit-button`) {
		t.Fatal("list view should not show pencil edit icons")
	}
	if strings.Contains(pageResponse.Body.String(), `<span class="field-name label">Name:</span>`) {
		t.Fatal("list view should hide field labels")
	}
	body := pageResponse.Body.String()
	linkStart := strings.Index(body, `href="/entry?`)
	if linkStart < 0 {
		t.Fatal("list entry does not link to its entry view")
	}
	hrefStart := linkStart + len(`href="`)
	hrefEnd := strings.Index(body[hrefStart:], `"`)
	if hrefEnd < 0 {
		t.Fatal("list entry link is unterminated")
	}
	entryURL := html.UnescapeString(body[hrefStart : hrefStart+hrefEnd])
	entryRequest := httptest.NewRequest(http.MethodGet, "http://example.test"+entryURL, nil)
	entryResponse := httptest.NewRecorder()
	handler.ServeHTTP(entryResponse, entryRequest)
	if entryResponse.Code != http.StatusOK {
		t.Fatalf("entry status = %d, want %d; body: %s", entryResponse.Code, http.StatusOK, entryResponse.Body.String())
	}
	if !strings.Contains(entryResponse.Body.String(), `field-edit-button`) {
		t.Fatal("entry view should retain pencil edit icons")
	}
	if !strings.Contains(entryResponse.Body.String(), `<span class="field-name label">Name:</span>`) {
		t.Fatal("entry view should show the title field")
	}
	if !strings.Contains(entryResponse.Body.String(), `<span class="field-name label">Email:</span>`) {
		t.Fatal("entry view should retain labels for non-title fields")
	}
	if strings.Contains(pageResponse.Body.String(), `hx-post="/edit"`) {
		t.Fatal("field edit forms should not be rendered until requested")
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
		`<span class="field-name label">Name:</span>`,
		`<form action="/edit" method="post"`,
		`hx-post="/edit"`,
		`hx-target="#workspace"`,
		`name="value" value="`,
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
	if !strings.Contains(viewResponse.Body.String(), `<output>`) || !strings.Contains(viewResponse.Body.String(), `field-edit-button`) {
		t.Fatalf("cancel response did not restore the static field view: %s", viewResponse.Body.String())
	}
	if strings.Contains(viewResponse.Body.String(), `<form action="/edit"`) {
		t.Fatal("cancel response unexpectedly contains an edit form")
	}

	formQuery.Set("field", "notAField")
	missingFieldRequest := httptest.NewRequest(http.MethodGet, "http://example.test/edit?"+formQuery.Encode(), nil)
	missingFieldRequest.Header.Set("HX-Request", "true")
	missingFieldResponse := httptest.NewRecorder()
	handler.ServeHTTP(missingFieldResponse, missingFieldRequest)
	if missingFieldResponse.Code != http.StatusNotFound {
		t.Fatalf("missing field form status = %d, want %d; body: %s", missingFieldResponse.Code, http.StatusNotFound, missingFieldResponse.Body.String())
	}
	if !strings.Contains(missingFieldResponse.Body.String(), "entry not found: path=core1.cue") {
		t.Fatalf("missing field form response does not identify the missing entry: %s", missingFieldResponse.Body.String())
	}

	formQuery.Set("file", "missing.cue")
	missingRequest := httptest.NewRequest(http.MethodGet, "http://example.test/edit?"+formQuery.Encode(), nil)
	missingRequest.Header.Set("HX-Request", "true")
	missingResponse := httptest.NewRecorder()
	handler.ServeHTTP(missingResponse, missingRequest)
	if missingResponse.Code != http.StatusNotFound {
		t.Fatalf("missing file form status = %d, want %d; body: %s", missingResponse.Code, http.StatusNotFound, missingResponse.Body.String())
	}
	if !strings.Contains(missingResponse.Body.String(), "file not found: path=missing.cue") {
		t.Fatalf("missing file form response does not identify the missing file: %s", missingResponse.Body.String())
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
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusInternalServerError, response.Body.String())
	}
}

func TestReadOnlyPageShowsStaticFieldsWithoutEditControls(t *testing.T) {
	handler, err := New(testSource())
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "http://example.test/?file=core1.cue", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}
	body := response.Body.String()
	if !strings.Contains(body, `<output>test1@testdomain.com</output>`) {
		t.Fatal("read-only page should show the static field value")
	}
	for _, unwanted := range []string{`field-edit-button`, `field-value-edit`, `hx-get="/edit`, `<form action="/edit"`} {
		if strings.Contains(body, unwanted) {
			t.Errorf("read-only page unexpectedly contains edit control %q", unwanted)
		}
	}
}

func TestWritableDirectoryCommitsEdits(t *testing.T) {
	tests := []struct {
		name         string
		htmx         bool
		wantStatus   int
		wantRedirect string
		fragment     bool
	}{
		{name: "browser form redirects to updated file list", wantStatus: http.StatusSeeOther, wantRedirect: "/?file=contacts.cue"},
		{name: "htmx form redirects to updated file list", htmx: true, wantStatus: http.StatusOK, wantRedirect: "/?file=contacts.cue", fragment: true},
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
			if got := response.Header().Get("HX-Redirect"); got != tt.wantRedirect {
				t.Errorf("HX-Redirect = %q, want %q", got, tt.wantRedirect)
			}
			if !tt.htmx {
				if got := response.Header().Get("Location"); got != tt.wantRedirect {
					t.Errorf("Location = %q, want %q", got, tt.wantRedirect)
				}
			}
			if tt.fragment {
				if !strings.Contains(response.Body.String(), `<main id="workspace"`) || strings.Contains(response.Body.String(), "<!doctype html>") {
					t.Fatalf("expected HTMX workspace fragment, got: %s", response.Body.String())
				}
			} else if !strings.Contains(response.Body.String(), "<!doctype html>") {
				t.Fatalf("expected full page response, got: %s", response.Body.String())
			}
			if !strings.Contains(response.Body.String(), "Saved Contact") {
				t.Errorf("updated value missing from response: %s", response.Body.String())
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

func TestEditCanUpdateDetailFieldsWithoutRemovingTheirMetadata(t *testing.T) {
	directory := t.TempDir()
	source := []byte(`[{Name: "Ada", Note: "before" @cuebook(detail)}]`)
	filePath := filepath.Join(directory, "people.cue")
	if err := os.WriteFile(filePath, source, 0o600); err != nil {
		t.Fatal(err)
	}
	handler, err := NewDirectory(directory)
	if err != nil {
		t.Fatal(err)
	}

	query := url.Values{"entry": {"0"}, "field": {"Note"}, "file": {"people.cue"}}
	formRequest := httptest.NewRequest(http.MethodGet, "http://example.test/edit?"+query.Encode(), nil)
	formRequest.Header.Set("HX-Request", "true")
	formResponse := httptest.NewRecorder()
	handler.ServeHTTP(formResponse, formRequest)
	if formResponse.Code != http.StatusOK || !strings.Contains(formResponse.Body.String(), `value="before"`) {
		t.Fatalf("detail edit form was not rendered: status = %d; body: %s", formResponse.Code, formResponse.Body.String())
	}

	response := submitEdit(t, handler, false, editValues("people.cue", "0", "Note", "after"))
	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusSeeOther, response.Body.String())
	}

	updated, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	book, err := cuebook.New(updated)
	if err != nil {
		t.Fatalf("saved source is invalid: %v", err)
	}
	entryValue, err := book.GetValue(0)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := cuebook.NewEntry(entryValue)
	if err != nil {
		t.Fatal(err)
	}
	field, ok := entry.GetFieldByName("Note")
	if !ok || field.String() != "after" {
		t.Fatalf("saved detail Note = %q, found = %t", field.String(), ok)
	}
	if len(entry.Details) != 1 || entry.Details[0].Name != "Note" {
		t.Fatalf("updated field is no longer a detail: %+v", entry.Details)
	}
}

func TestEditCanPopulateAbsentOptionalDetailFields(t *testing.T) {
	directory := t.TempDir()
	source := []byte(`#person: {Name: string, Note?: string @cuebook(detail)}
[...#person] & [{Name: "Ada"}]`)
	filePath := filepath.Join(directory, "people.cue")
	if err := os.WriteFile(filePath, source, 0o600); err != nil {
		t.Fatal(err)
	}
	handler, err := NewDirectory(directory)
	if err != nil {
		t.Fatal(err)
	}

	response := submitEdit(t, handler, false, editValues("people.cue", "0", "Note", "added"))
	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusSeeOther, response.Body.String())
	}

	updated, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	book, err := cuebook.New(updated)
	if err != nil {
		t.Fatalf("saved source is invalid: %v", err)
	}
	entryValue, err := book.GetValue(0)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := cuebook.NewEntry(entryValue)
	if err != nil {
		t.Fatal(err)
	}
	field, ok := entry.GetFieldByName("Note")
	if !ok || field.String() != "added" {
		t.Fatalf("saved detail Note = %q, found = %t", field.String(), ok)
	}
	if len(entry.Details) != 1 || entry.Details[0].Name != "Note" {
		t.Fatalf("populated field is not a detail: %+v", entry.Details)
	}
}

func TestEditRedirectUsesMountedRoutePrefix(t *testing.T) {
	handler, err := NewWithCommitter(testSource(), &recordingCommitter{}, WithServeMuxPrefix("/catalog"))
	if err != nil {
		t.Fatal(err)
	}

	values := editValues("core1.cue", "0", "Name", "Updated")
	request := httptest.NewRequest(http.MethodPost, "http://example.test/catalog/edit", strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Origin", "http://example.test")
	request.Header.Set("HX-Request", "true")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if got, want := response.Header().Get("HX-Redirect"), "/catalog/?file=core1.cue"; got != want {
		t.Errorf("HX-Redirect = %q, want %q", got, want)
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
		htmx       bool
		wantStatus int
		wantNotice string
	}{
		{
			name:       "invalid email is rejected by cue validation",
			values:     editValues("core1.cue", "0", "Email", "not-an-email"),
			origin:     "http://example.test",
			wantStatus: http.StatusInternalServerError,
			wantNotice: "does not satisfy the CUE constraints",
		},
		{
			name:       "htmx validation error uses the default status",
			values:     editValues("core1.cue", "0", "Email", "not-an-email"),
			origin:     "http://example.test",
			htmx:       true,
			wantStatus: http.StatusInternalServerError,
			wantNotice: "does not satisfy the CUE constraints",
		},
		{
			name:       "missing field is not found",
			values:     editValues("core1.cue", "0", "notAField", "value"),
			origin:     "http://example.test",
			wantStatus: http.StatusNotFound,
			wantNotice: "entry not found: path=core1.cue",
		},
		{
			name:       "out of range entry is not found",
			values:     editValues("core1.cue", "9", "Name", "value"),
			origin:     "http://example.test",
			wantStatus: http.StatusNotFound,
			wantNotice: "entry not found: path=core1.cue",
		},
		{
			name:       "path traversal is not found",
			values:     editValues("../core1.cue", "0", "Name", "value"),
			origin:     "http://example.test",
			wantStatus: http.StatusNotFound,
			wantNotice: "file not found: path=../core1.cue",
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
			response := submitEditWithOrigin(t, handler, tt.htmx, tt.values, tt.origin)
			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", response.Code, tt.wantStatus, response.Body.String())
			}
			if !strings.Contains(response.Body.String(), tt.wantNotice) {
				t.Errorf("body does not contain %q", tt.wantNotice)
			}
			if tt.origin == "http://example.test" {
				if got := response.Header().Get("HX-Trigger"); !strings.Contains(got, tt.wantNotice) {
					t.Errorf("HX-Trigger = %q, want error flash %q", got, tt.wantNotice)
				}
				if got := response.Header().Get("HX-Redirect"); got != "" {
					t.Errorf("HX-Redirect = %q, want no redirect on error", got)
				}
				if !strings.Contains(response.Body.String(), `class="notice notification is-danger" role="alert">`) {
					t.Errorf("error flash is not rendered as an alert: %s", response.Body.String())
				}
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
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusInternalServerError, response.Body.String())
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
