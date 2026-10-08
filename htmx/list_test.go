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

func TestIndexEntryLinksOpenMatchingEntry(t *testing.T) {
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
			if !strings.Contains(body, `<div class="grid is-col-min-16 is-gap-2 entries-grid">`) {
				t.Fatalf("entry list is not wrapped in the Bulma grid: %s", body)
			}
			if !strings.Contains(body, `<article class="entry card cell"`) {
				t.Fatalf("entry card is not a Bulma grid cell: %s", body)
			}
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
			entryHref := html.UnescapeString(body[hrefStart : hrefStart+hrefLength])
			parsedHref, err := url.Parse(entryHref)
			if err != nil {
				t.Fatalf("parse generated entry href %q: %v", entryHref, err)
			}
			if parsedHref.Path != "/entry" {
				t.Errorf("entry href path = %q, want /entry", parsedHref.Path)
			}
			query := parsedHref.Query()
			if got := query.Get("path"); got != test.filePath {
				t.Errorf("entry href path query = %q, want %q", got, test.filePath)
			}
			if got := query.Get("head"); got != strconv.Itoa(selectedRange.Head) {
				t.Errorf("entry href head query = %q, want %d", got, selectedRange.Head)
			}
			if got := query.Get("tail"); got != strconv.Itoa(selectedRange.Tail) {
				t.Errorf("entry href tail query = %q, want %d", got, selectedRange.Tail)
			}

			entryRequest := httptest.NewRequest(http.MethodGet, "http://example.test"+entryHref, nil)
			entryResponse := httptest.NewRecorder()
			handler.ServeHTTP(entryResponse, entryRequest)
			if entryResponse.Code != http.StatusOK {
				t.Fatalf("entry status = %d, want %d; body: %s", entryResponse.Code, http.StatusOK, entryResponse.Body.String())
			}
			entryBody := entryResponse.Body.String()
			for _, want := range []string{"Second entry", "second@example.test"} {
				if !strings.Contains(entryBody, want) {
					t.Errorf("entry body does not contain %q: %s", want, entryBody)
				}
			}
			for _, unwanted := range []string{"First entry", "first@example.test"} {
				if strings.Contains(entryBody, unwanted) {
					t.Errorf("entry body unexpectedly contains %q: %s", unwanted, entryBody)
				}
			}
		})
	}
}

func TestEntryTitleFieldIsOmittedFromEntryContent(t *testing.T) {
	tests := []struct {
		name            string
		source          string
		title           string
		wantContent     []string
		wantTitleCounts int
	}{
		{
			name: "annotated title field is hidden while matching field values remain",
			source: `#person: {
	Title: string @cuebook(title)
	Alias: string
	Email: string
}
[...#person] & [{Title: "Ada Title", Alias: "Ada Title", Email: "ada@example.test"}]`,
			title:           "Ada Title",
			wantContent:     []string{"Ada Title", "ada@example.test"},
			wantTitleCounts: 1,
		},
		{
			name:            "fallback title field is hidden",
			source:          `[{Name: "Fallback title"}]`,
			title:           "Fallback title",
			wantTitleCounts: 0,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			handler, err := New(fstest.MapFS{
				"people.cue": &fstest.MapFile{Data: []byte(test.source)},
			})
			if err != nil {
				t.Fatal(err)
			}

			request := httptest.NewRequest(http.MethodGet, "http://example.test/?file=people.cue", nil)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
			}

			body := response.Body.String()
			if !strings.Contains(body, test.title) {
				t.Fatalf("entry title %q is missing: %s", test.title, body)
			}
			contentStart := strings.Index(body, `<div class="card-content entry-content">`)
			if contentStart < 0 {
				t.Fatalf("entry content is missing: %s", body)
			}
			contentEnd := strings.Index(body[contentStart:], `</article>`)
			if contentEnd < 0 {
				t.Fatalf("entry article is not closed: %s", body[contentStart:])
			}
			content := body[contentStart : contentStart+contentEnd]
			if got := strings.Count(content, test.title); got != test.wantTitleCounts {
				t.Errorf("title appears %d times in entry content, want %d: %s", got, test.wantTitleCounts, content)
			}
			for _, want := range test.wantContent {
				if !strings.Contains(content, want) {
					t.Errorf("entry content does not contain %q: %s", want, content)
				}
			}
		})
	}
}

func TestFileFrontmatterView(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		fileName      string
		source        string
		wantTitle     string
		wantVisible   string
		wantCollapsed string
		wantComponent bool
	}{
		{
			name:          "frontmatter details after thematic break are collapsible",
			fileName:      "described.cue",
			source:        "// Document title\n//\n// Description with **markup**.\n//\n// ---\n// Additional details.\n[{Name: \"entry\"}]\n",
			wantTitle:     "Document title",
			wantVisible:   "Description with <strong>markup</strong>.",
			wantCollapsed: "Additional details.",
			wantComponent: true,
		},
		{
			name:        "description without thematic break remains visible",
			fileName:    "plain-description.cue",
			source:      "// Plain title\n//\n// Plain description.\n[{Name: \"entry\"}]\n",
			wantTitle:   "Plain title",
			wantVisible: "Plain description.",
		},
		{
			name:      "filename fallback when title is absent",
			fileName:  "plain.cue",
			source:    `[{Name: "entry"}]`,
			wantTitle: "plain.cue",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			handler, err := New(fstest.MapFS{
				test.fileName: &fstest.MapFile{Data: []byte(test.source)},
			})
			if err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest(http.MethodGet, "http://example.test/?file="+test.fileName, nil)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
			}

			body := response.Body.String()
			if !strings.Contains(body, `<h1 class="document-title title is-4">`+test.wantTitle+`</h1>`) {
				t.Errorf("file title %q not shown: %s", test.wantTitle, body)
			}
			if test.wantVisible != "" && !strings.Contains(body, test.wantVisible) {
				t.Errorf("visible description %q not shown: %s", test.wantVisible, body)
			}
			componentStart := strings.Index(body, `<remember-details data-storage-key="view-file-frontmatter">`)
			if !test.wantComponent {
				if componentStart >= 0 {
					t.Errorf("unexpected frontmatter details component: %s", body)
				}
				return
			}
			if componentStart < 0 {
				t.Fatalf("frontmatter details component is absent: %s", body)
			}
			componentEndOffset := strings.Index(body[componentStart:], `</remember-details>`)
			if componentEndOffset < 0 {
				t.Fatalf("frontmatter details component is not closed: %s", body)
			}
			componentEnd := componentStart + componentEndOffset
			component := body[componentStart:componentEnd]
			if !strings.Contains(component, "<summary>Description</summary>") ||
				!strings.Contains(component, `data-details-content`) ||
				!strings.Contains(component, test.wantCollapsed) {
				t.Errorf("description component missing expected content: %s", component)
			}
			if titlePosition := strings.Index(body, test.wantTitle); titlePosition > componentStart {
				t.Errorf("title should appear before the collapsible description: %s", body)
			}
			if visiblePosition := strings.Index(body, test.wantVisible); visiblePosition > componentStart {
				t.Errorf("visible description should precede the collapsed details: %s", body)
			}
		})
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
			contains:   []string{"core1.cue", "subfolder", "sub1.cue", `<span>core1</span>`, `<span>sub1</span>`, `data-file="core1.cue"`, `data-file="subfolder/sub1.cue"`, `href="/?file=subfolder%2Fsub1.cue"`, `<file-tree-node class="file-tree-node" data-node-path="subfolder">`, `class="tree-folder-icon"`, `class="tree-file-icon"`, `class="tree-children"`, `<link rel="icon" type="image/svg+xml" href="/assets/favicon.svg">`, "bulma.css", "htmx-2.0.4.min.js", "/assets/theme.js", "/assets/file-tree.js", `data-theme="dark"`, `id="theme-toggle"`, `aria-pressed="true"`, `data-theme-icon="moon" style="display: inline-block"`, `d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"`, `data-theme-icon="sun" style="display: none"`, `<circle cx="12" cy="12" r="4"/>`},
			omits:      []string{"notes.txt", `<span>core1.cue</span>`, `<span>sub1.cue</span>`, "First11111aa", "Dark mode", "Light mode"},
		},
		{
			name:       "selected nested file renders entries",
			method:     http.MethodGet,
			path:       "/?file=subfolder%2Fsub1.cue",
			wantStatus: http.StatusOK,
			contains:   []string{"subfolder/sub1.cue", "First11111aa", "test1@testdomain.com", "Read-only source", `aria-current="page"`, `<article class="entry card cell" id="entry-0" data-entry-index="0"`, `<header class="card-header">`, `<div class="card-content entry-content">`, `<h2 class="card-header-title">`},
			omits:      []string{`hx-post="/add"`, `data-entry-drag-handle`, `entry-drag-handle`},
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
			contains:   []string{"file not found: path=../secret.cue"},
		},
		{
			name:       "missing file is not found",
			method:     http.MethodGet,
			path:       "/?file=missing.cue",
			wantStatus: http.StatusNotFound,
			contains:   []string{"file not found: path=missing.cue"},
		},
		{
			name:       "invalid cue is reported safely",
			method:     http.MethodGet,
			path:       "/?file=invalid.cue",
			wantStatus: http.StatusInternalServerError,
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

func TestArchiveFilesAppearInSeparateNavigationSection(t *testing.T) {
	tests := []struct {
		name     string
		source   fstest.MapFS
		selected string
		contains []string
		omits    []string
	}{
		{
			name: "archive files are listed separately and selected archive expands section",
			source: fstest.MapFS{
				"regular.cue":             &fstest.MapFile{Data: []byte(`[{Name: "Regular"}]`)},
				".archive/2025-01-01.cue": &fstest.MapFile{Data: []byte(`[{Name: "Archived"}]`)},
			},
			selected: ".archive/2025-01-01.cue",
			contains: []string{
				`data-file="regular.cue"`,
				`<remember-details data-storage-key="archive-files" open>`,
				"<summary>Archive</summary>",
				`data-file=".archive/2025-01-01.cue"`,
				`aria-current="page"`,
			},
			omits: []string{`data-node-path=".archive"`},
		},
		{
			name: "archive-only source does not show archive as a root folder or claim there are no files",
			source: fstest.MapFS{
				".archive/2025-01-01.cue": &fstest.MapFile{Data: []byte(`[{Name: "Archived"}]`)},
			},
			contains: []string{
				"<summary>Archive</summary>",
				`data-file=".archive/2025-01-01.cue"`,
			},
			omits: []string{`data-node-path=".archive"`, "No CUE files found."},
		},
		{
			name: "empty archive section is still available",
			source: fstest.MapFS{
				"regular.cue": &fstest.MapFile{Data: []byte(`[{Name: "Regular"}]`)},
			},
			contains: []string{"<summary>Archive</summary>", "No archived files."},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			handler, err := New(test.source)
			if err != nil {
				t.Fatal(err)
			}

			target := "/"
			if test.selected != "" {
				target += "?file=" + url.QueryEscape(test.selected)
			}
			request := httptest.NewRequest(http.MethodGet, "http://example.test"+target, nil)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
			}

			body := response.Body.String()
			for _, want := range test.contains {
				if !strings.Contains(body, want) {
					t.Errorf("body does not contain %q", want)
				}
			}
			for _, unwanted := range test.omits {
				if strings.Contains(body, unwanted) {
					t.Errorf("body unexpectedly contains %q", unwanted)
				}
			}
		})
	}
}
