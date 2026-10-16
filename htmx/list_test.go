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
			if !strings.Contains(body, `<div class="grid is-gap-2 entries-grid">`) {
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
			contentStart := strings.Index(body, `<div class="card-content">`)
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
		name              string
		fileName          string
		source            string
		wantBreadcrumb    string
		wantMetadataTitle string
		wantVisible       string
		wantCollapsed     string
		wantComponent     bool
	}{
		{
			name:              "frontmatter details after thematic break are collapsible",
			fileName:          "described.cue",
			source:            "// Document title\n//\n// Description with **markup**.\n//\n// ---\n// Additional details.\n[{Name: \"entry\"}]\n",
			wantBreadcrumb:    "described",
			wantMetadataTitle: "Document title",
			wantVisible:       "Description with <strong>markup</strong>.",
			wantCollapsed:     "Additional details.",
			wantComponent:     true,
		},
		{
			name:              "description without thematic break remains visible",
			fileName:          "plain-description.cue",
			source:            "// Plain title\n//\n// Plain description.\n[{Name: \"entry\"}]\n",
			wantBreadcrumb:    "plain-description",
			wantMetadataTitle: "Plain title",
			wantVisible:       "Plain description.",
		},
		{
			name:           "no metadata title leaves only the basename breadcrumb",
			fileName:       "plain.cue",
			source:         `[{Name: "entry"}]`,
			wantBreadcrumb: "plain",
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
			for _, want := range []string{
				`<nav class="breadcrumb" aria-label="File location">`,
				`href="/" hx-get="/" hx-target="#workspace"`,
				`title="` + test.fileName + `">` + test.wantBreadcrumb + `</a>`,
			} {
				if !strings.Contains(body, want) {
					t.Errorf("file breadcrumb or title is missing %q: %s", want, body)
				}
			}
			metadataTitleHeading := `<h1 class="document-title title is-4">` + test.wantMetadataTitle + `</h1>`
			if test.wantMetadataTitle != "" {
				metadataTitlePosition := strings.Index(body, metadataTitleHeading)
				if metadataTitlePosition < 0 {
					t.Errorf("metadata title heading %q is missing: %s", test.wantMetadataTitle, body)
				} else if breadcrumbPosition := strings.Index(body, `<nav class="breadcrumb" aria-label="File location">`); breadcrumbPosition < 0 || metadataTitlePosition >= breadcrumbPosition {
					t.Errorf("metadata title should appear above the breadcrumb: %s", body)
				}
			} else if strings.Contains(body, `<h1 class="document-title title is-4">`) {
				t.Errorf("title heading should be absent without metadata: %s", body)
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
			if !strings.Contains(component, "<summary>More...</summary>") ||
				!strings.Contains(component, `data-details-content`) ||
				!strings.Contains(component, test.wantCollapsed) {
				t.Errorf("description component missing expected content: %s", component)
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
			contains:   []string{"core1.cue", "subfolder", "sub1.cue", `<span>core1</span>`, `<span>sub1</span>`, `data-file="core1.cue"`, `data-file="subfolder/sub1.cue"`, `href="/?file=subfolder%2Fsub1.cue"`, `<file-tree-node class="file-tree-node" data-node-path="subfolder">`, `class="tree-folder-icon"`, `class="tree-file-icon"`, `class="tree-children"`, `<link rel="icon" type="image/svg+xml" href="/assets/favicon.svg">`, "bulma.css", "htmx-2.0.4.min.js", "/assets/theme.js", "/assets/file-tree.js", `data-theme="dark"`, `id="theme-toggle"`, `aria-pressed="true"`, `<button id="theme-toggle" class="button has-text-info" type="button" aria-label="Toggle color theme" aria-pressed="true">`, `☼`},
			omits:      []string{"notes.txt", `<span>core1.cue</span>`, `<span>sub1.cue</span>`, "First11111aa", "Dark mode", "Light mode"},
		},
		{
			name:       "selected nested file renders entries and file breadcrumbs",
			method:     http.MethodGet,
			path:       "/?file=subfolder%2Fsub1.cue",
			wantStatus: http.StatusOK,
			contains:   []string{"subfolder/sub1.cue", "First11111aa", "test1@testdomain.com", "Read-only source", `<nav class="breadcrumb" aria-label="File location">`, `href="/?folder=subfolder"`, `aria-current="page"`, `<article class="entry card cell" id="entry-0" data-entry-index="0"`, `<header class="card-header">`, `<div class="card-content">`, `<h2 class="card-header-title">`},
			omits:      []string{`hx-post="/add"`, `data-entry-drag-handle`, `entry-drag-handle`},
		},
		{
			name:       "htmx request gets workspace fragment with file breadcrumb",
			method:     http.MethodGet,
			path:       "/?file=core1.cue",
			htmx:       true,
			wantStatus: http.StatusOK,
			contains:   []string{`<main id="workspace"`, `<nav class="breadcrumb" aria-label="File location">`, `href="/"`, "First11111aa"},
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

func TestFolderViewListsFilesAndFolders(t *testing.T) {
	handler, err := New(fstest.MapFS{
		"root.cue":              &fstest.MapFile{Data: []byte(`[{Name: "Root"}]`)},
		"nested/first.cue":      &fstest.MapFile{Data: []byte(`[{Name: "First"}]`)},
		"nested/child/deep.cue": &fstest.MapFile{Data: []byte(`[{Name: "Deep"}]`)},
		"other.cue":             &fstest.MapFile{Data: []byte(`[{Name: "Other"}]`)},
	})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		folder   string
		htmx     bool
		contains []string
		omits    []string
	}{
		{
			name: "root folder lists direct files and subfolders",
			contains: []string{
				`<nav class="breadcrumb" aria-label="Folder location">`,
				`<li class="is-active" aria-current="page"><span>Files</span></li>`,
				`data-folder-path="nested"`,
				`href="/?folder=nested"`,
				`data-file="root.cue"`,
			},
			omits: []string{`data-file="nested/first.cue"`, `data-file="nested/child/deep.cue"`},
		},
		{
			name:   "nested folder shows its contents and breadcrumb",
			folder: "nested",
			htmx:   true,
			contains: []string{
				`<li><a href="/" hx-get="/"`,
				`<li class="is-active" aria-current="page"><span>nested</span></li>`,
				`data-file="nested/first.cue"`,
				`data-folder-path="nested/child"`,
			},
			omits: []string{`data-file="root.cue"`, `data-file="nested/child/deep.cue"`},
		},
		{
			name:   "deep folder breadcrumb links to its parent",
			folder: "nested/child",
			contains: []string{
				`href="/?folder=nested"`,
				`<li class="is-active" aria-current="page"><span>child</span></li>`,
				`data-file="nested/child/deep.cue"`,
			},
			omits: []string{`data-file="nested/first.cue"`},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			query := url.Values{}
			if test.folder != "" {
				query.Set("folder", test.folder)
			}
			target := "/"
			if encoded := query.Encode(); encoded != "" {
				target += "?" + encoded
			}
			request := httptest.NewRequest(http.MethodGet, "http://example.test"+target, nil)
			if test.htmx {
				request.Header.Set("HX-Request", "true")
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
			}

			body := response.Body.String()
			if test.htmx && !strings.HasPrefix(body, `<main id="workspace"`) {
				t.Fatalf("HTMX folder request did not return the workspace fragment: %s", body)
			}
			if test.htmx && strings.Contains(body, `<!doctype html>`) {
				t.Fatalf("HTMX folder request unexpectedly returned a full page: %s", body)
			}
			start := strings.Index(body, `<section class="folder-view"`)
			if start < 0 {
				t.Fatalf("folder view is missing: %s", body)
			}
			endOffset := strings.Index(body[start:], `</section>`)
			if endOffset < 0 {
				t.Fatalf("folder view is not closed: %s", body[start:])
			}
			folderView := body[start : start+endOffset+len(`</section>`)]
			for _, want := range test.contains {
				if !strings.Contains(folderView, want) {
					t.Errorf("folder view does not contain %q: %s", want, folderView)
				}
			}
			for _, unwanted := range test.omits {
				if strings.Contains(folderView, unwanted) {
					t.Errorf("folder view unexpectedly contains %q: %s", unwanted, folderView)
				}
			}
		})
	}
}

func TestFolderViewRejectsUnknownFolder(t *testing.T) {
	t.Parallel()

	handler, err := New(fstest.MapFS{
		"nested/file.cue": &fstest.MapFile{Data: []byte(`[{Name: "Entry"}]`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "http://example.test/?folder=..%2Fprivate", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusNotFound, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "not found: path=../private") {
		t.Fatalf("response does not identify the missing folder: %s", response.Body.String())
	}
}

func TestArchiveShortcutOpensFolderView(t *testing.T) {
	tests := []struct {
		name        string
		source      fstest.MapFS
		path        string
		contains    []string
		omits       []string
		navContains []string
		navOmits    []string
	}{
		{
			name: "selected archive file highlights shortcut but is not listed in the sidebar",
			source: fstest.MapFS{
				"regular.cue":             &fstest.MapFile{Data: []byte(`[{Name: "Regular"}]`)},
				".archive/2025-01-01.cue": &fstest.MapFile{Data: []byte(`[{Name: "Archived"}]`)},
			},
			path:        "/?file=.archive%2F2025-01-01.cue",
			navContains: []string{`href="/?folder=.archive"`, `hx-get="/?folder=.archive"`, `aria-current="page"`},
			navOmits:    []string{`data-node-path=".archive"`, `data-file=".archive/2025-01-01.cue"`},
		},
		{
			name: "archive folder view lists archived files",
			source: fstest.MapFS{
				"regular.cue":             &fstest.MapFile{Data: []byte(`[{Name: "Regular"}]`)},
				".archive/2025-01-01.cue": &fstest.MapFile{Data: []byte(`[{Name: "Archived"}]`)},
			},
			path:        "/?folder=.archive",
			contains:    []string{`data-file=".archive/2025-01-01.cue"`, `<span>2025-01-01</span>`},
			omits:       []string{`data-file="regular.cue"`},
			navContains: []string{`href="/?folder=.archive"`, `aria-current="page"`},
			navOmits:    []string{`data-file=".archive/2025-01-01.cue"`},
		},
		{
			name: "archive-only source keeps archive out of the regular sidebar tree",
			source: fstest.MapFS{
				".archive/2025-01-01.cue": &fstest.MapFile{Data: []byte(`[{Name: "Archived"}]`)},
			},
			path:        "/",
			contains:    []string{`data-folder-path=".archive"`},
			navContains: []string{`href="/?folder=.archive"`},
			navOmits:    []string{`data-node-path=".archive"`, `data-file=".archive/2025-01-01.cue"`},
		},
		{
			name: "archive shortcut opens an empty archive folder",
			source: fstest.MapFS{
				"regular.cue": &fstest.MapFile{Data: []byte(`[{Name: "Regular"}]`)},
			},
			path:        "/?folder=.archive",
			contains:    []string{"This folder contains no CUE files or subfolders."},
			navContains: []string{`href="/?folder=.archive"`, `aria-current="page"`},
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
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "http://example.test"+test.path, nil))
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
			}
			body := response.Body.String()
			navStart := strings.Index(body, `<nav class="file-nav menu"`)
			if navStart < 0 {
				t.Fatalf("file navigation is missing: %s", body)
			}
			navEnd := strings.Index(body[navStart:], `</nav>`)
			if navEnd < 0 {
				t.Fatalf("file navigation is not closed: %s", body[navStart:])
			}
			navigation := body[navStart : navStart+navEnd+len(`</nav>`)]
			for _, want := range test.navContains {
				if !strings.Contains(navigation, want) {
					t.Errorf("navigation does not contain %q: %s", want, navigation)
				}
			}
			for _, unwanted := range test.navOmits {
				if strings.Contains(navigation, unwanted) {
					t.Errorf("navigation unexpectedly contains %q: %s", unwanted, navigation)
				}
			}
			if strings.Contains(test.path, "folder=") {
				start := strings.Index(body, `<section class="folder-view"`)
				if start < 0 {
					t.Fatalf("folder view is missing: %s", body)
				}
				endOffset := strings.Index(body[start:], `</section>`)
				if endOffset < 0 {
					t.Fatalf("folder view is not closed: %s", body[start:])
				}
				body = body[start : start+endOffset+len(`</section>`)]
			}
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
