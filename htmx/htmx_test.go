package htmx

import (
	"context"
	"embed"
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
		`name="entry.0.field" value="Name"`,
		`name="entry.1.field" value="Email"`,
		`name="entry.2.field" value="Notes"`,
		`name="entry.3.field" value="Password"`,
		"(optional)",
		`<remember-details data-storage-key="add-entry-optional">`,
		"<summary>Optional fields</summary>",
		`<div id="add-entry-optional-content" data-details-content>`,
		`<script src="/assets/remember-details.js" defer></script>`,
		"Add entry",
	} {
		if !strings.Contains(response.Body.String(), want) {
			t.Errorf("page does not contain %q", want)
		}
	}

	body := response.Body.String()
	componentStart := strings.Index(body, `<remember-details data-storage-key="add-entry-optional">`)
	if componentStart < 0 {
		t.Fatalf("optional fields are not wrapped by the persistent details component: %s", body)
	}
	closingTag := `</remember-details>`
	closingOffset := strings.Index(body[componentStart:], closingTag)
	if closingOffset < 0 {
		t.Fatalf("optional fields component is not closed: %s", body)
	}
	componentEnd := componentStart + closingOffset
	optionalMarkup := body[componentStart:componentEnd]
	for _, want := range []string{`name="entry.2.field" value="Notes"`, `name="entry.3.field" value="Password"`} {
		if !strings.Contains(optionalMarkup, want) {
			t.Errorf("persistent optional fields do not contain %q: %s", want, optionalMarkup)
		}
	}
	for _, notWant := range []string{`name="entry.0.field" value="Name"`, `name="entry.1.field" value="Email"`} {
		if strings.Contains(optionalMarkup, notWant) {
			t.Errorf("required field %q is inside the optional fields component: %s", notWant, optionalMarkup)
		}
	}
}

func TestAddEntryFormStartsAtTypeZeroValues(t *testing.T) {
	t.Parallel()

	source := []byte(`#entry: {
	Name: *"Default name" | string
	Count: *42 | (int & >=10)
	Enabled: *true | bool
}
[...#entry] & [{Name: "Existing", Count: 12, Enabled: true}]
`)
	committer := &recordingCommitter{}
	handler, err := NewWithCommitter(fstest.MapFS{
		"defaults.cue": &fstest.MapFile{Data: source},
	}, committer)
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
		`name="entry.0.field" value="Name"`,
		`name="entry.0.value" value="" required`,
		`name="entry.1.field" value="Count"`,
		`type="number" name="entry.1.value" value="0" step="any" required`,
		`name="entry.2.field" value="Enabled"`,
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

	failure := submitAdd(t, handler, true, addEntryValues("defaults.cue", [][2]string{
		{"Name", "Retained name"}, {"Count", "7"}, {"Enabled", "true"},
	}))
	if failure.Code != http.StatusInternalServerError {
		t.Fatalf("failure status = %d, want %d; body: %s", failure.Code, http.StatusInternalServerError, failure.Body.String())
	}
	if !strings.Contains(failure.Body.String(), "does not satisfy the CUE constraints") {
		t.Fatalf("validation error missing from response: %s", failure.Body.String())
	}
	if committer.calls != 0 {
		t.Fatalf("committer calls = %d, want 0 for rejected entry", committer.calls)
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
		{name: "version-pinned Bulma", path: "/assets/bulma.css", contains: []string{"bulma.io v1.0.4", ".grid.is-col-min-16", ".is-gap-2{gap:1rem!important}"}, wantStatus: http.StatusOK},
		{name: "Bulma license", path: "/assets/bulma-LICENSE.txt", contains: []string{"The MIT License"}, wantStatus: http.StatusOK},
		{name: "SVG book favicon", path: "/assets/favicon.svg", contains: []string{"<svg", `fill="#22c55e"`, `fill="#3b82f6"`}, wantStatus: http.StatusOK},
		{name: "htmx license", path: "/assets/htmx-LICENSE.txt", contains: []string{"Zero-Clause BSD"}, wantStatus: http.StatusOK},
		{name: "stylesheet", path: "/assets/app.css", contains: []string{"grid-template-columns", "grid-template-columns: subgrid", ".entry-content {", ".entries-grid", ".entries-grid .entry.card", "--bulma-grid-column-min: min(100%, 24rem)", "max-width: 40rem", "justify-self: center", "width: 100%", "text-align: right", "text-decoration: underline dotted", ".entry.drop-before::before", ".entry.drop-after::after", "file-tree-node", ".tree-chevron", ".tree-file-link.is-drop-target", "remember-details > summary", "remember-details > [data-details-content][hidden]", ".tree-children[hidden]", ".delete-confirmation-card", ".move-confirmation-card", "prefers-reduced-motion"}, wantStatus: http.StatusOK},
		{name: "file tree component", path: "/assets/file-tree.js", contains: []string{"cuebook-file-tree-folded", `customElements.define("file-tree-node"`, "readFoldedPaths", "writeFoldedPaths", `setAttribute("aria-expanded"`, "syncCurrentFile", `htmx:pushedIntoHistory`}, wantStatus: http.StatusOK},
		{name: "entry move controller", path: "/assets/entry-move.js", contains: []string{"data-entry-drag-handle", "await window.htmx.ajax(\"POST\", `${routePrefix}/move`, {", "drop-before", "drop-after", "destination", "is-drop-target", "cuebook:move-confirmation-request"}, wantStatus: http.StatusOK},
		{name: "move confirmation controller", path: "/assets/move-confirm.js", contains: []string{`getElementById("move-confirmation")`, "data-move-confirmation-entry", "sourceCard.dataset.file === destinationLink.dataset.file", "event.preventDefault()", `event.key === "Escape"`}, wantStatus: http.StatusOK},
		{name: "remember details component", path: "/assets/remember-details.js", contains: []string{`customElements.define("remember-details"`, "localStorage.getItem", "localStorage.setItem", "dataset.storageKey", "aria-expanded", "keydown"}, wantStatus: http.StatusOK},
		{name: "delete confirmation controller", path: "/assets/delete-confirm.js", contains: []string{`form[action$="/delete"]`, `event.preventDefault()`, `event.stopImmediatePropagation()`, "textContent", `aria-hidden`, "requestSubmit", `event.key === "Escape"`}, wantStatus: http.StatusOK},
		{name: "theme controller", path: "/assets/theme.js", contains: []string{"localStorage.setItem", `querySelectorAll("[data-theme-icon]")`, `? "moon" : "sun"`, `icon.style.display`, `? "inline-block" : "none"`}, wantStatus: http.StatusOK},
		{name: "live reload component", path: "/assets/live-reload.js", contains: []string{"class LiveReload extends HTMLElement", `new EventSource(eventsURL)`, "window.location.reload()"}, wantStatus: http.StatusOK},
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

func TestPageIncludesLiveReloadComponent(t *testing.T) {
	handler, err := New(fstest.MapFS{})
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}
	for _, want := range []string{
		`<script src="/assets/live-reload.js" defer></script>`,
		`<live-reload data-events-url="/events"></live-reload>`,
	} {
		if !strings.Contains(response.Body.String(), want) {
			t.Errorf("page does not contain %q", want)
		}
	}
}

func TestLiveReloadEventStreamStartsAndStopsOnContextCancellation(t *testing.T) {
	handler, err := New(fstest.MapFS{})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	request := httptest.NewRequest(http.MethodGet, "http://example.test/events", nil).WithContext(ctx)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if got := response.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Errorf("Content-Type = %q, want text/event-stream", got)
	}
	if !response.Flushed {
		t.Error("SSE connection was not flushed")
	}
	if got := response.Body.String(); got != ": connected\n\n" {
		t.Errorf("initial SSE data = %q, want %q", got, ": connected\n\n")
	}
}

func TestWritableDirectoryAppendsEntries(t *testing.T) {
	tests := []struct {
		name string
		htmx bool
	}{
		{name: "browser form receives updated page"},
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
				if response.Code != http.StatusOK {
					t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
				}
				if !strings.Contains(response.Body.String(), "<!doctype html>") || !strings.Contains(response.Body.String(), "Added Contact") {
					t.Fatalf("expected updated full page, got: %s", response.Body.String())
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
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "<!doctype html>") || !strings.Contains(response.Body.String(), "First Contact") {
		t.Fatalf("expected updated full page, got: %s", response.Body.String())
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
		htmx       bool
		wantStatus int
		wantNotice string
	}{
		{
			name: "invalid email is rejected by CUE",
			values: addEntryValues("core1.cue", [][2]string{
				{"Name", "Invalid Contact"}, {"Email", "not-an-email"},
			}),
			origin:     "http://example.test",
			htmx:       true,
			wantStatus: http.StatusInternalServerError,
			wantNotice: "does not satisfy the CUE constraints",
		},
		{
			name: "unknown field is rejected",
			values: addEntryValues("core1.cue", [][2]string{
				{"Name", "Contact"}, {"Email", "valid@example.test"}, {"notAField", "value"},
			}),
			origin:     "http://example.test",
			wantStatus: http.StatusInternalServerError,
			wantNotice: "unknown field",
		},
		{
			name:       "missing required field is rejected",
			values:     addEntryValues("core1.cue", [][2]string{{"Name", "Contact"}}),
			origin:     "http://example.test",
			wantStatus: http.StatusInternalServerError,
			wantNotice: "required entry field is missing",
		},
		{
			name: "duplicate field is rejected",
			values: addEntryValues("core1.cue", [][2]string{
				{"Name", "Contact"}, {"Name", "Other Contact"}, {"Email", "valid@example.test"},
			}),
			origin:     "http://example.test",
			wantStatus: http.StatusInternalServerError,
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
			response := submitAddWithOrigin(t, handler, tt.htmx, tt.values, tt.origin)
			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", response.Code, tt.wantStatus, response.Body.String())
			}
			body := response.Body.String()
			if !strings.Contains(body, tt.wantNotice) {
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
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusInternalServerError, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "This source is read-only.") {
		t.Fatal("expected read-only explanation")
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
	for index, field := range fields {
		prefix := "entry." + strconv.Itoa(index) + "."
		values.Set(prefix+"field", field[0])
		values.Set(prefix+"value", field[1])
	}
	return values
}

type recordingCommitter struct {
	calls int
	err   error
}

func (c *recordingCommitter) Commit(_ string, _ patch.Patch) error {
	c.calls++
	return c.err
}
