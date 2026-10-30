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

func TestAddEntryFormRendersFieldDocumentationAsHelp(t *testing.T) {
	t.Parallel()

	source := []byte(`#person: {
	// Display <name> & alias.
	Name: string
	// Optional contact address.
	Email?: string
	Active: bool
}
[...#person] & []`)
	handler, err := NewWithCommitter(fstest.MapFS{
		"people.cue": &fstest.MapFile{Data: source},
	}, &recordingCommitter{})
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
	for _, want := range []string{
		`<p class="help">Display &lt;name&gt; &amp; alias.</p>`,
		`<p class="help">Optional contact address.</p>`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("add form is missing help text %q: %s", want, body)
		}
	}
	if got := strings.Count(body, `<p class="help">`); got != 2 {
		t.Errorf("add form has %d help descriptions, want 2", got)
	}
}

func TestEntryViewHidesFileMetadataAndShowsFileBreadcrumb(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		file      string
		fileTitle string
		prefix    string
		htmx      bool
		readOnly  bool
	}{
		{name: "browser navigation", file: "people.cue"},
		{name: "HTMX navigation", file: "people.cue", htmx: true},
		{name: "entry view hides file metadata", file: "people.cue", fileTitle: "Document title"},
		{name: "mounted nested file with special characters", file: "nested/people & friends.cue", fileTitle: "Friends", prefix: "/catalog", htmx: true},
		{name: "read only entry", file: "people.cue", readOnly: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			sourceText := `[{Name: "Target entry"}]`
			if tt.fileTitle != "" {
				sourceText = "// " + tt.fileTitle + "\n//\n// Metadata description.\n" + sourceText
			}
			source := []byte(sourceText)
			book, err := cuebook.New(source)
			if err != nil {
				t.Fatal(err)
			}
			entry, err := book.GetValue(0)
			if err != nil {
				t.Fatal(err)
			}
			span, err := cuebook.NewByteRange(entry)
			if err != nil {
				t.Fatal(err)
			}
			files := fstest.MapFS{tt.file: &fstest.MapFile{Data: source}}
			var handler http.Handler
			if tt.readOnly {
				handler, err = New(files, WithServeMuxPrefix(tt.prefix))
			} else {
				handler, err = NewWithCommitter(files, &recordingCommitter{}, WithServeMuxPrefix(tt.prefix))
			}
			if err != nil {
				t.Fatal(err)
			}
			get := func(target string) string {
				t.Helper()
				request := httptest.NewRequest(http.MethodGet, "http://example.test"+target, nil)
				if tt.htmx {
					request.Header.Set("HX-Request", "true")
				}
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, request)
				if response.Code != http.StatusOK {
					t.Fatalf("GET %s: status %d; body: %s", target, response.Code, response.Body.String())
				}
				return response.Body.String()
			}
			body := get(tt.prefix + entryURL(tt.file, span))
			start := strings.Index(body, `<nav class="breadcrumb" aria-label="File location">`)
			if start < 0 {
				t.Fatalf("file breadcrumb not found: %s", body)
			}
			end := strings.Index(body[start:], `</nav>`)
			if end < 0 {
				t.Fatalf("file breadcrumb is not closed: %s", body)
			}
			heading := html.UnescapeString(body[start : start+end+len(`</nav>`)])
			wantLabel := tt.file
			if slash := strings.LastIndex(wantLabel, "/"); slash >= 0 {
				wantLabel = wantLabel[slash+1:]
			}
			wantLabel = strings.TrimSuffix(wantLabel, ".cue")
			if wantLabel == "" {
				wantLabel = tt.file
			}
			wantURL := tt.prefix + fileURL(tt.file)
			for _, want := range []string{
				`<nav class="breadcrumb" aria-label="File location">`,
				`href="` + wantURL + `"`, `hx-get="` + wantURL + `"`,
				`title="` + tt.file + `">` + wantLabel + `</a>`,
				`hx-target="#workspace"`, `hx-swap="outerHTML"`,
			} {
				if !strings.Contains(heading, want) {
					t.Errorf("entry title row missing %q: %s", want, heading)
				}
			}
			if strings.Contains(heading, `<li class="is-active" aria-current="page">`) {
				t.Errorf("entry breadcrumb marks the file link as the current non-clickable page: %s", heading)
			}
			backLink := `<a href="` + wantURL + `" class="button" type="button is-primary" title="Back">Back</a>`
			if !strings.Contains(html.UnescapeString(body), backLink) {
				t.Errorf("entry back link does not point to parent file %q: %s", wantURL, body)
			}
			pathParts := strings.Split(tt.file, "/")
			for index, directory := range pathParts[:len(pathParts)-1] {
				directoryPath := strings.Join(pathParts[:index+1], "/")
				wantDirectoryURL := tt.prefix + folderURL(directoryPath)
				if !strings.Contains(heading, `href="`+wantDirectoryURL+`"`) || !strings.Contains(heading, directory) {
					t.Errorf("file breadcrumb does not link directory %q to its folder view %q: %s", directory, wantDirectoryURL, heading)
				}
			}
			if strings.Contains(body, `<h1 class="title document-title is-4">`) {
				t.Errorf("entry view unexpectedly displays file metadata title: %s", body)
			}
			if strings.Contains(body, "Metadata description.") {
				t.Errorf("entry view unexpectedly displays file metadata description: %s", body)
			}
			list := get(wantURL)
			if tt.fileTitle != "" {
				metadataHeading := `<h1 class="title document-title is-4">` + tt.fileTitle + `</h1>`
				if !strings.Contains(list, metadataHeading) || !strings.Contains(list, "Metadata description.") {
					t.Errorf("file list should display its metadata title and description: %s", list)
				}
			}
			if strings.Contains(list, `aria-label="Return to file `) {
				t.Error("file list unexpectedly includes a return-to-file button")
			}
			if !strings.Contains(list, "Target entry") {
				t.Error("file breadcrumb did not render the source file's entries")
			}
		})
	}
}

func TestListAndEntryUseSeparateEntryTemplates(t *testing.T) {
	t.Parallel()

	const filePath = "people.cue"
	source := []byte(`[{Name: "Target entry"}]`)
	book, err := cuebook.New(source)
	if err != nil {
		t.Fatal(err)
	}
	var entryRange cuebook.ByteRange
	for entry, err := range book.EachEntry() {
		if err != nil {
			t.Fatal(err)
		}
		entryRange, err = cuebook.NewByteRange(entry.Value)
		if err != nil {
			t.Fatal(err)
		}
		break
	}

	handler, err := NewWithCommitter(fstest.MapFS{
		filePath: &fstest.MapFile{Data: source},
	}, &recordingCommitter{})
	if err != nil {
		t.Fatal(err)
	}

	listQuery := url.Values{"file": {filePath}}
	listRequest := httptest.NewRequest(http.MethodGet, "http://example.test/?"+listQuery.Encode(), nil)
	listResponse := httptest.NewRecorder()
	handler.ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d; body: %s", listResponse.Code, http.StatusOK, listResponse.Body.String())
	}
	if strings.Contains(listResponse.Body.String(), `action="/archive"`) {
		t.Errorf("list view unexpectedly includes the archive form: %s", listResponse.Body.String())
	}

	entryRequest := httptest.NewRequest(http.MethodGet, "http://example.test"+entryURL(filePath, entryRange), nil)
	entryResponse := httptest.NewRecorder()
	handler.ServeHTTP(entryResponse, entryRequest)
	if entryResponse.Code != http.StatusOK {
		t.Fatalf("entry status = %d, want %d; body: %s", entryResponse.Code, http.StatusOK, entryResponse.Body.String())
	}
	entryBody := entryResponse.Body.String()
	if !strings.Contains(entryBody, "<!doctype html>") {
		t.Errorf("normal entry view should render the full page: %s", entryBody)
	}
	archiveFormIndex := strings.Index(entryBody, `action="/archive"`)
	articleEndIndex := strings.Index(entryBody, `</article>`)
	if archiveFormIndex == -1 {
		t.Errorf("entry view does not include the archive form: %s", entryBody)
	} else if articleEndIndex == -1 || archiveFormIndex < articleEndIndex {
		t.Errorf("archive form is not placed below the entry container: %s", entryBody)
	}
	if !strings.Contains(entryBody, `<div class="buttons entry-actions is-centered">`) || !strings.Contains(entryBody, `>Archive</button>`) {
		t.Errorf("entry view does not render the archive action group: %s", entryBody)
	}
	if !strings.Contains(entryBody, `<remember-details class="add-entry-dock" data-storage-key="add-entry-dock">`) || !strings.Contains(entryBody, `action="/add"`) {
		t.Errorf("full entry view does not include the add-entry dock: %s", entryBody)
	}

	fragmentRequest := httptest.NewRequest(http.MethodGet, "http://example.test"+entryURL(filePath, entryRange), nil)
	fragmentRequest.Header.Set("HX-Request", "true")
	fragmentResponse := httptest.NewRecorder()
	handler.ServeHTTP(fragmentResponse, fragmentRequest)
	if fragmentResponse.Code != http.StatusOK {
		t.Fatalf("entry fragment status = %d, want %d; body: %s", fragmentResponse.Code, http.StatusOK, fragmentResponse.Body.String())
	}
	fragmentBody := fragmentResponse.Body.String()
	if strings.Contains(fragmentBody, "<!doctype html>") || !strings.Contains(fragmentBody, `<remember-details class="add-entry-dock" data-storage-key="add-entry-dock">`) || !strings.Contains(fragmentBody, `action="/add"`) {
		t.Errorf("entry fragment does not include the add-entry dock as a workspace fragment: %s", fragmentBody)
	}
}

func TestEntryViewAlwaysShowsDetailsAndEmptyOptionalFields(t *testing.T) {
	t.Parallel()

	const filePath = "people.cue"
	source := []byte(`#person: {Name: string, Nick?: string, Note?: string @cuebook(detail)}
[...#person] & [{Name: "Ada"}]`)
	book, err := cuebook.New(source)
	if err != nil {
		t.Fatal(err)
	}
	var entryRange cuebook.ByteRange
	for entry, err := range book.EachEntry() {
		if err != nil {
			t.Fatal(err)
		}
		entryRange, err = cuebook.NewByteRange(entry.Value)
		if err != nil {
			t.Fatal(err)
		}
		break
	}

	handler, err := New(fstest.MapFS{
		filePath: &fstest.MapFile{Data: source},
	})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "http://example.test"+entryURL(filePath, entryRange), nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}

	body := response.Body.String()
	articleStart := strings.Index(body, `<article `)
	articleEnd := strings.Index(body, `</article>`)
	if articleStart < 0 || articleEnd < articleStart {
		t.Fatalf("entry article not found: %s", body)
	}
	article := body[articleStart:articleEnd]
	for _, want := range []string{`Ada`, `Nick`, `Note:`, `<output>...</output>`} {
		if !strings.Contains(article, want) {
			t.Errorf("entry view does not contain %q: %s", want, article)
		}
	}
	if strings.Contains(article, "remember-details") || strings.Contains(article, "<details") {
		t.Errorf("entry details are hidden behind a disclosure component: %s", article)
	}
}

func TestEntryHandlerRendersOnlyTheEntryMatchingItsByteRange(t *testing.T) {
	source := []byte(`[
	{Name: "First entry", Email: "first@example.test"},
	{Name: "Second entry", Email: "second@example.test", Active: true},
]`)
	book, err := cuebook.New(source)
	if err != nil {
		t.Fatal(err)
	}

	var secondRange cuebook.ByteRange
	index := 0
	for entry, err := range book.EachEntry() {
		if err != nil {
			t.Fatal(err)
		}
		if index == 1 {
			secondRange, err = cuebook.NewByteRange(entry.Value)
			if err != nil {
				t.Fatal(err)
			}
			break
		}
		index++
	}

	const filePath = "people.cue"
	handler, err := New(fstest.MapFS{
		filePath: &fstest.MapFile{Data: source},
	})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		path       string
		head       string
		tail       string
		wantStatus int
		contains   []string
		omits      []string
	}{
		{
			name:       "renders matching entry as a single entry fragment",
			path:       filePath,
			head:       strconv.Itoa(secondRange.Head),
			tail:       strconv.Itoa(secondRange.Tail),
			wantStatus: http.StatusOK,
			contains:   []string{`data-entry-index="1"`, `data-file="people.cue"`, "Second entry", "second@example.test"},
			omits:      []string{"First entry", "first@example.test", "<!doctype html>"},
		},
		{
			name:       "missing file path is rejected",
			head:       strconv.Itoa(secondRange.Head),
			tail:       strconv.Itoa(secondRange.Tail),
			wantStatus: http.StatusInternalServerError,
			contains:   []string{"The entry request is invalid."},
		},
		{
			name:       "invalid range is rejected",
			path:       filePath,
			head:       "not-an-integer",
			tail:       strconv.Itoa(secondRange.Tail),
			wantStatus: http.StatusInternalServerError,
			contains:   []string{"The entry request is invalid."},
		},
		{
			name:       "unmatched range is not found",
			path:       filePath,
			head:       "0",
			tail:       "1",
			wantStatus: http.StatusNotFound,
			contains:   []string{"entry not found: path=people.cue byteRange=0-1"},
		},
		{
			name:       "unknown file is not found",
			path:       "missing.cue",
			head:       strconv.Itoa(secondRange.Head),
			tail:       strconv.Itoa(secondRange.Tail),
			wantStatus: http.StatusNotFound,
			contains:   []string{"file not found: path=missing.cue"},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			query := url.Values{}
			if test.path != "" {
				query.Set("path", test.path)
			}
			if test.head != "" {
				query.Set("head", test.head)
			}
			if test.tail != "" {
				query.Set("tail", test.tail)
			}

			request := httptest.NewRequest(http.MethodGet, "http://example.test/entry?"+query.Encode(), nil)
			request.Header.Set("HX-Request", "true")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", response.Code, test.wantStatus, response.Body.String())
			}

			body := response.Body.String()
			for _, want := range test.contains {
				if !strings.Contains(body, want) {
					t.Errorf("body does not contain %q: %s", want, body)
				}
			}
			for _, unwanted := range test.omits {
				if strings.Contains(body, unwanted) {
					t.Errorf("body unexpectedly contains %q: %s", unwanted, body)
				}
			}
			if test.wantStatus == http.StatusOK && strings.Count(body, `<article `) != 1 {
				t.Errorf("rendered entry count = %d, want 1: %s", strings.Count(body, `<article `), body)
			}
		})
	}
}
