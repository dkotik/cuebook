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

func TestAddEntryFormShowsAllSchemaFieldsImmediately(t *testing.T) {
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

	body := response.Body.String()
	for _, want := range []string{
		`<remember-details class="add-entry-dock" data-storage-key="add-entry-dock">`,
		`<summary aria-controls="add-entry-dock-panel">✚</summary>`,
		`<section class="add-entry" id="add-entry-dock-panel" aria-labelledby="add-entry-heading" data-details-content>`,
		`action="/add"`,
		`hx-post="/add"`,
		`<h2 id="add-entry-heading" class="title is-5">Add entry</h2>`,

		`name="entry.0.field" value="Name"`,
		`name="entry.1.field" value="Email"`,
		`name="entry.2.field" value="Notes"`,
		`name="entry.3.field" value="Password"`,
		`placeholder="..."`,
		"(optional)",
		"Add entry",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("page does not contain %q", want)
		}
	}
	formStart := strings.Index(body, `<section class="add-entry`)
	if formStart < 0 {
		t.Fatalf("add-entry form not found: %s", body)
	}
	formEnd := strings.Index(body[formStart:], `</section>`)
	if formEnd < 0 {
		t.Fatalf("add-entry form is not closed: %s", body)
	}
	if !strings.Contains(body[formStart+formEnd:], `</remember-details>`) {
		t.Errorf("add-entry panel is not inside its dock: %s", body[formStart:])
	}

}

func TestAddEntryFormUsesCurrentFile(t *testing.T) {
	t.Parallel()

	const source = `#entry: {Name: string}
[...#entry] & [{Name: "Existing"}]
`
	files := fstest.MapFS{
		"root.cue":         &fstest.MapFile{Data: []byte(source)},
		"nested/other.cue": &fstest.MapFile{Data: []byte(source)},
		"ignored.txt":      &fstest.MapFile{Data: []byte("not CUE")},
	}
	tests := []struct {
		name     string
		file     string
		entry    bool
		fragment bool
	}{
		{name: "root list", file: "root.cue"},
		{name: "nested list", file: "nested/other.cue"},
		{name: "root entry", file: "root.cue", entry: true},
		{name: "nested entry", file: "nested/other.cue", entry: true},
		{name: "HTMX entry", file: "nested/other.cue", entry: true, fragment: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			handler, err := NewWithCommitter(files, &recordingCommitter{})
			if err != nil {
				t.Fatal(err)
			}
			target := "/?" + url.Values{"file": {tt.file}}.Encode()
			if tt.entry {
				book, err := cuebook.New([]byte(source))
				if err != nil {
					t.Fatal(err)
				}
				value, err := book.GetValue(0)
				if err != nil {
					t.Fatal(err)
				}
				entryRange, err := cuebook.NewByteRange(value)
				if err != nil {
					t.Fatal(err)
				}
				target = entryURL(tt.file, entryRange)
			}
			request := httptest.NewRequest(http.MethodGet, "http://example.test"+target, nil)
			if tt.fragment {
				request.Header.Set("HX-Request", "true")
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
			}

			body := response.Body.String()
			formStart := strings.Index(body, `<form id="add-entry-form"`)
			if formStart < 0 {
				t.Fatal("add-entry form is missing")
			}
			formEnd := strings.Index(body[formStart:], `</form>`)
			if formEnd < 0 {
				t.Fatal("add-entry form is not closed")
			}
			form := body[formStart : formStart+formEnd]
			for _, want := range []string{
				`<input type="hidden" name="file" value="` + tt.file + `">`,
				`name="entry.0.field"`,
				`hx-post="/add"`,
			} {
				if !strings.Contains(form, want) {
					t.Errorf("add-entry form missing %q", want)
				}
			}
			if strings.Contains(body, "add-entry-file") || strings.Contains(form, "data-file-picker-open") {
				t.Error("add-entry form still exposes a file picker")
			}
		})
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
	if !strings.Contains(body, `<remember-details class="add-entry-dock" data-storage-key="add-entry-dock">`) {
		t.Fatalf("add-entry dock does not persist its disclosure state: %s", body)
	}
	formStart := strings.Index(body, `<form id="add-entry-form" action="/add"`)
	if formStart < 0 {
		t.Fatalf("add-entry form not found: %s", body)
	}
	formEnd := strings.Index(body[formStart:], `</form>`)
	if formEnd < 0 {
		t.Fatalf("add-entry form is not closed: %s", body[formStart:])
	}
	form := body[formStart : formStart+formEnd]
	if !strings.Contains(form, `<div id="add-entry-error" class="form-error" aria-live="assertive"></div>`) {
		t.Errorf("add-entry error target is not inside the form: %s", form)
	}
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
	if failure.Header().Get("HX-Retarget") != "#add-entry-error" {
		t.Errorf("HX-Retarget = %q, want #add-entry-error", failure.Header().Get("HX-Retarget"))
	}
	if failure.Header().Get("HX-Reswap") != "innerHTML" {
		t.Errorf("HX-Reswap = %q, want innerHTML", failure.Header().Get("HX-Reswap"))
	}
	failureBody := failure.Body.String()
	if !strings.Contains(failureBody, "does not satisfy the CUE constraints") {
		t.Fatalf("validation error missing from response: %s", failureBody)
	}
	if !strings.Contains(failureBody, `<p class="notice notification is-danger" role="alert">`) || strings.Contains(failureBody, `add-entry-error`) || strings.Contains(failureBody, `<form`) || strings.Contains(failureBody, `<main id="workspace"`) {
		t.Fatalf("HTMX validation response should be a generic error fragment, not replace the form or workspace: %s", failureBody)
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
		{name: "entry color themes", path: "/assets/theme.css", contains: []string{`:root[data-theme="dark"]`, "--entry-card-background: #181825", `:root[data-theme="light"]`, "--entry-card-header-background: #cdd6f4"}, wantStatus: http.StatusOK},
		{name: "Bulma license", path: "/assets/bulma-LICENSE.txt", contains: []string{"The MIT License"}, wantStatus: http.StatusOK},
		{name: "SVG book favicon", path: "/assets/favicon.svg", contains: []string{"<svg", `fill="#22c55e"`, `fill="#3b82f6"`}, wantStatus: http.StatusOK},
		{name: "htmx license", path: "/assets/htmx-LICENSE.txt", contains: []string{"Zero-Clause BSD"}, wantStatus: http.StatusOK},
		{name: "stylesheet", path: "/assets/app.css", contains: []string{"grid-template-columns", "grid-template-columns: subgrid", ".entry-content {", ".entries-grid", ".entries-grid .entry.card", "--bulma-grid-column-min: min(100%, 12rem)", "max-width: 40rem", "justify-self: center", "width: 100%", "text-align: right", "text-decoration: underline dotted", ".entry.drop-before::before", ".entry.drop-after::after", "file-tree-node", ".tree-chevron", ".tree-file-link.is-drop-target", "remember-details > summary", "remember-details > [data-details-content][hidden]", ".add-entry-dock:not([open])", ".add-entry-dock > summary::before", ".tree-children[hidden]", ".archive-confirmation-card", ".move-confirmation-card", ".file-selector-dialog", ".file-picker-tree", ".file-picker-card", "prefers-reduced-motion"}, wantStatus: http.StatusOK},
		{name: "file tree component", path: "/assets/file-tree.js", contains: []string{"cuebook-file-tree-folded", `customElements.define("file-tree-node"`, "readFoldedPaths", "writeFoldedPaths", `setAttribute("aria-expanded"`, "syncCurrentFile", `document.querySelectorAll("a.tree-file-link")`, `htmx:pushedIntoHistory`}, wantStatus: http.StatusOK},
		{name: "custom search element", path: "/assets/search-control.js", contains: []string{`customElements.define("cuebook-search"`, `this.input.value = ""`, "updateButtons()", "this.form.insertBefore(this.clearControl, this.inputControl)", "this.clearControl.remove()", "this.input.focus()"}, wantStatus: http.StatusOK},
		{name: "entry move controller", path: "/assets/entry-move.js", contains: []string{"data-entry-drag-handle", "await window.htmx.ajax(\"POST\", `${routePrefix}/move`, {", "drop-before", "drop-after", "destination", "is-drop-target", "cuebook:move-confirmation-request"}, wantStatus: http.StatusOK},

		{name: "file picker controller", path: "/assets/file-picker.js", contains: []string{"data-file-picker-open", "dialog.showModal()", "data-file-picker-confirm", "input.value = selected.value", "label.textContent = selected.value", "form.closest(\"dialog[data-file-picker-dialog]\")?.close()"}, wantStatus: http.StatusOK},
		{name: "move confirmation controller", path: "/assets/move-confirm.js", contains: []string{`getElementById("move-confirmation")`, "data-move-confirmation-entry", "sourceCard.dataset.file === destinationLink.dataset.file", "event.preventDefault()", `event.key === "Escape"`}, wantStatus: http.StatusOK},
		{name: "remember details component", path: "/assets/remember-details.js", contains: []string{`customElements.define("remember-details"`, "localStorage.getItem", "localStorage.setItem", "dataset.storageKey", "aria-expanded", "keydown"}, wantStatus: http.StatusOK},
		{name: "archive confirmation controller", path: "/assets/archive-confirm.js", contains: []string{`form[action$="/archive"]`, `form[hx-post$="/archive"]`, `event.preventDefault()`, `event.stopImmediatePropagation()`, "textContent", `aria-hidden`, "requestSubmit", `event.key === "Escape"`}, wantStatus: http.StatusOK},
		{name: "theme controller", path: "/assets/theme.js", contains: []string{"localStorage.setItem", `querySelectorAll("[data-theme-icon]")`, `? "moon" : "sun"`, `icon.style.display`, `? "inline-block" : "none"`}, wantStatus: http.StatusOK},
		{name: "navbar controller", path: "/assets/navbar.js", contains: []string{`classList.toggle("is-active", expanded)`, `setAttribute("aria-expanded", String(expanded))`, `document.querySelector(".topbar-search")?.addEventListener("submit"`}, wantStatus: http.StatusOK},
		{name: "entry copy controller", path: "/assets/entry-copy.js", contains: []string{`addEventListener("click"`, `navigator.clipboard.writeText(source.textContent)`, `"cuebook:flash"`}, wantStatus: http.StatusOK},
		{name: "entry paste controller", path: "/assets/entry-paste.js", contains: []string{`addEventListener("paste"`, `getData("text/plain")`, `window.htmx.ajax("POST"`, "isContentEditable"}, wantStatus: http.StatusOK},
		{name: "flash message controller", path: "/assets/flash.js", contains: []string{`"cuebook:flash"`, "textContent", "flash.hidden = false"}, wantStatus: http.StatusOK},
		{name: "field edit dialog controller", path: "/assets/field-edit-dialog.js", contains: []string{"htmx:afterSwap", `classList.add("is-active")`, `event.key === "Escape"`, `source.closest("#field-edit-modal form")`, `modalSavePending`, `last.focus()`}, wantStatus: http.StatusOK},
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

func TestPageIncludesFixedTopNavbarAndLiveReloadComponent(t *testing.T) {
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
	page := response.Body.String()
	for _, want := range []string{
		`<body class="has-navbar-fixed-top" data-route-prefix="">`,
		`<nav class="topbar navbar is-fixed-top" role="navigation" aria-label="main navigation">`,
		`aria-expanded="false" aria-controls="cuebook-navbar-menu" data-target="cuebook-navbar-menu"`,
		`<script src="/assets/navbar.js" defer></script>`,
		`<script src="/assets/live-reload.js" defer></script>`,
		`<live-reload data-events-url="/events"></live-reload>`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("page does not contain %q", want)
		}
	}

	brandStart := strings.Index(page, `<div class="navbar-brand">`)
	searchStart := strings.Index(page, `<div class="navbar-item topbar-search-item is-expanded">`)
	burgerStart := strings.Index(page, `<button class="navbar-burger"`)
	menuStart := strings.Index(page, `<div id="cuebook-navbar-menu"`)
	brandEnd := -1
	if burgerStart >= 0 {
		if offset := strings.Index(page[burgerStart:], `</div>`); offset >= 0 {
			brandEnd = burgerStart + offset
		}
	}
	if !(brandStart >= 0 && brandStart < searchStart && searchStart < burgerStart && burgerStart < brandEnd && brandEnd < menuStart) {
		t.Errorf("search should remain in the visible navbar brand row outside the collapsible menu")
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
				body := response.Body.String()
				wantEntryID := `id="entry-` + strconv.Itoa(beforeLength) + `"`
				if strings.Contains(body, `<main id="workspace"`) || strings.Contains(body, `entries-grid`) ||
					!strings.Contains(body, `<article class="card" `+wantEntryID) ||
					!strings.Contains(body, `class="buttons entry-actions"`) || !strings.Contains(body, "Added Contact") {
					t.Fatalf("expected the newly-added entry view fragment, got: %s", body)
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
			name: "missing file is not found",
			values: addEntryValues("missing.cue", [][2]string{
				{"Name", "Contact"}, {"Email", "valid@example.test"},
			}),
			origin:     "http://example.test",
			wantStatus: http.StatusNotFound,
			wantNotice: "file not found: path=missing.cue",
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
