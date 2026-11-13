package htmx

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"cuelang.org/go/cue"
	"github.com/dkotik/cuebook"
)

const identifiedHTMXSource = `#entry: {
	ID: string @cuebook(uuid)
	Name: string @cuebook(title)
	Email: string
	...
}
[...#entry] & [
	{ID: "first", Name: "First", Email: "first@example.test"},
	{ID: "target", Name: "Target", Email: "target@example.test"},
]`

func TestEntryIndexByUUIDRequiresUniqueIdentifier(t *testing.T) {
	book, err := cuebook.New([]byte(identifiedHTMXSource))
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name      string
		uuid      string
		wantIndex int
		wantError error
	}{
		{name: "unique identifier", uuid: "target", wantIndex: 1},
		{name: "missing identifier", uuid: "missing", wantError: cuebook.ErrBookEntryNotFound},
		{name: "empty identifier", wantError: cuebook.ErrBookEntryNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := entryIndexByUUID(book, tt.uuid)
			if tt.wantError != nil {
				if !errors.Is(err, tt.wantError) {
					t.Fatalf("entryIndexByUUID() error = %v, want %v", err, tt.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("entryIndexByUUID() error = %v", err)
			}
			if got != tt.wantIndex {
				t.Errorf("entryIndexByUUID() = %d, want %d", got, tt.wantIndex)
			}
		})
	}

	duplicateSource := []byte(`#entry: {ID: string @cuebook(uuid), Name: string}
[...#entry] & [{ID: "duplicate", Name: "First"}, {ID: "duplicate", Name: "Second"}]`)
	duplicateBook, err := cuebook.New(duplicateSource)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entryIndexByUUID(duplicateBook, "duplicate"); !errors.Is(err, cuebook.ErrPatchTargetAmbiguous) {
		t.Fatalf("duplicate identifier error = %v, want %v", err, cuebook.ErrPatchTargetAmbiguous)
	}
}

func TestUUIDEntryRoutesAndGeneratedActionsAreSeparate(t *testing.T) {
	t.Parallel()

	files := fstest.MapFS{"people.cue": &fstest.MapFile{Data: []byte(identifiedHTMXSource)}}
	handler, err := newTestHandler(files, mapFileSystem{files: files})
	if err != nil {
		t.Fatal(err)
	}

	listResponse := serveGET(t, handler, "/?file=people.cue")
	if !strings.Contains(listResponse.Body.String(), `href="/entry/uuid/people?uuid=target"`) {
		t.Fatalf("identified entry link does not use the UUID route: %s", listResponse.Body.String())
	}

	entryResponse := serveGET(t, handler, "/entry/uuid/people?uuid=target")
	body := entryResponse.Body.String()
	if !strings.Contains(body, "Target") || strings.Contains(body, "First") {
		t.Fatalf("UUID entry route did not select only the requested entry: %s", body)
	}
	for _, want := range []string{
		`action="/archive/uuid"`,
		`name="uuid" value="target"`,
		`action="/move/uuid"`,
		`name="from_uuid" value="target"`,
		`action="/add-field/uuid"`,
		`hx-get="/edit/uuid?field=Email&amp;file=people.cue&amp;uuid=target&amp;view=entry"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("identified entry action missing %q: %s", want, body)
		}
	}

	legacyRequest := httptest.NewRequest(http.MethodGet, "http://example.test/entry?path=people.cue&uuid=target", nil)
	legacyResponse := httptest.NewRecorder()
	handler.ServeHTTP(legacyResponse, legacyRequest)
	if legacyResponse.Code == http.StatusOK {
		t.Fatalf("legacy range route accepted a UUID selector: %s", legacyResponse.Body.String())
	}

	editForm := serveGET(t, handler, "/edit/uuid?file=people.cue&uuid=target&field=Email&view=entry")
	for _, want := range []string{
		`action="/edit/uuid"`,
		`name="uuid" value="target"`,
	} {
		if !strings.Contains(editForm.Body.String(), want) {
			t.Errorf("UUID edit form missing %q: %s", want, editForm.Body.String())
		}
	}
	if strings.Contains(editForm.Body.String(), `name="entry"`) {
		t.Fatalf("UUID edit form includes a positional selector: %s", editForm.Body.String())
	}
}

func TestUUIDEntryHandlerRejectsMissingOrDuplicateIdentifiers(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		source string
		uuid   string
		want   string
	}{
		{
			name:   "missing identifier",
			source: identifiedHTMXSource,
			uuid:   "missing",
			want:   "entry not found",
		},
		{
			name: "duplicate identifier",
			source: `#entry: {ID: string @cuebook(uuid), Name: string}
[...#entry] & [{ID: "duplicate", Name: "First"}, {ID: "duplicate", Name: "Second"}]`,
			uuid: "duplicate",
			want: "patch target is ambiguous",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := fstest.MapFS{"people.cue": &fstest.MapFile{Data: []byte(tt.source)}}
			handler, err := New(files)
			if err != nil {
				t.Fatal(err)
			}
			response := serveGET(t, handler, "/entry/uuid/people?uuid="+url.QueryEscape(tt.uuid))
			if response.Code == http.StatusOK || !strings.Contains(strings.ToLower(response.Body.String()), tt.want) {
				t.Fatalf("UUID route status/body = %d/%q, want error containing %q", response.Code, response.Body.String(), tt.want)
			}
		})
	}
}

func TestUUIDEditHandlerTargetsIdentifierRoute(t *testing.T) {
	t.Parallel()

	files := fstest.MapFS{"people.cue": &fstest.MapFile{Data: []byte(identifiedHTMXSource)}}
	handler, err := newTestHandler(files, mapFileSystem{files: files})
	if err != nil {
		t.Fatal(err)
	}
	values := url.Values{
		"file":  {"people.cue"},
		"uuid":  {"target"},
		"field": {"Email"},
		"value": {"updated@example.test"},
		"view":  {"entry"},
	}
	response := servePOST(t, handler, "/edit/uuid", values)
	if response.Code != http.StatusOK {
		t.Fatalf("UUID edit status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if got, want := response.Header().Get("HX-Replace-Url"), "/entry/uuid/people?uuid=target"; got != want {
		t.Errorf("replacement URL = %q, want %q", got, want)
	}

	book, err := cuebook.New(files["people.cue"].Data)
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		index int
		want  string
	}{
		{index: 0, want: "first@example.test"},
		{index: 1, want: "updated@example.test"},
	} {
		entry, err := book.GetValueByIndex(tt.index)
		if err != nil {
			t.Fatal(err)
		}
		got, err := entry.LookupPath(cue.ParsePath("Email")).String()
		if err != nil || got != tt.want {
			t.Errorf("entry %d Email = %q, err = %v; want %q", tt.index, got, err, tt.want)
		}
	}

	legacyValues := url.Values{
		"file":  {"people.cue"},
		"uuid":  {"target"},
		"field": {"Email"},
		"value": {"must-not-be-applied@example.test"},
	}
	legacyResponse := servePOST(t, handler, "/edit", legacyValues)
	if legacyResponse.Code == http.StatusOK {
		t.Fatalf("legacy edit route accepted a UUID selector: %s", legacyResponse.Body.String())
	}
}

func TestUUIDNewFieldHandlerTargetsIdentifierRoute(t *testing.T) {
	t.Parallel()

	files := fstest.MapFS{"people.cue": &fstest.MapFile{Data: []byte(identifiedHTMXSource)}}
	handler, err := newTestHandler(files, mapFileSystem{files: files})
	if err != nil {
		t.Fatal(err)
	}
	values := url.Values{
		"file":         {"people.cue"},
		"uuid":         {"target"},
		"name":         {"Role"},
		"kind":         {"string"},
		"value_string": {"maintainer"},
	}
	response := servePOST(t, handler, "/add-field/uuid", values)
	if response.Code != http.StatusOK {
		t.Fatalf("UUID add-field status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if got, want := response.Header().Get("HX-Replace-Url"), "/entry/uuid/people?uuid=target"; got != want {
		t.Errorf("replacement URL = %q, want %q", got, want)
	}
	book, err := cuebook.New(files["people.cue"].Data)
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		index int
		want  string
	}{
		{index: 0, want: ""},
		{index: 1, want: "maintainer"},
	} {
		entry, err := book.GetValueByIndex(tt.index)
		if err != nil {
			t.Fatal(err)
		}
		role := entry.LookupPath(cue.ParsePath("Role"))
		if !role.Exists() {
			if tt.want != "" {
				t.Errorf("entry %d has no Role field; want %q", tt.index, tt.want)
			}
			continue
		}
		got, err := role.String()
		if err != nil || got != tt.want {
			t.Errorf("entry %d Role = %q, err = %v; want %q", tt.index, got, err, tt.want)
		}
	}
}

func TestUUIDArchiveHandlerTargetsIdentifierRoute(t *testing.T) {
	t.Parallel()

	files := fstest.MapFS{"people.cue": &fstest.MapFile{Data: []byte(identifiedHTMXSource)}}
	storage := mapFileSystem{files: files}
	handler, err := newTestHandler(files, storage)
	if err != nil {
		t.Fatal(err)
	}
	waitForSearchResponse(t, handler, "archive")
	response := servePOST(t, handler, "/archive/uuid", url.Values{
		"file": {"people.cue"},
		"uuid": {"target"},
	})
	if response.Code != http.StatusOK {
		t.Fatalf("UUID archive status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}

	sourceBook, err := cuebook.New(files["people.cue"].Data)
	if err != nil {
		t.Fatal(err)
	}
	if length, err := sourceBook.Len(); err != nil || length != 1 {
		t.Fatalf("source entry count = %d, err = %v; want 1", length, err)
	}
	remaining, err := sourceBook.GetEntryByUUID("first")
	if err != nil || remaining.Title != "First" {
		t.Fatalf("remaining entry = %+v, err = %v; want first entry", remaining, err)
	}

	archiveName := cuebook.ArchiveFileName(time.Now())
	archiveFile, ok := files[archiveName]
	if !ok {
		var names []string
		for name := range files {
			names = append(names, name)
		}
		t.Fatalf("archive file %q is missing; files = %v", archiveName, names)
	}
	archiveBook, err := cuebook.New(archiveFile.Data)
	if err != nil {
		t.Fatal(err)
	}
	archived, err := archiveBook.GetValueByIndex(0)
	if err != nil {
		t.Fatal(err)
	}
	archivedID, idErr := archived.LookupPath(cue.ParsePath("ID")).String()
	archivedName, nameErr := archived.LookupPath(cue.ParsePath("Name")).String()
	if idErr != nil || nameErr != nil || archivedID != "target" || archivedName != "Target" {
		t.Fatalf("archived entry ID/Name = %q/%q, errors = %v/%v; want target/Target", archivedID, archivedName, idErr, nameErr)
	}
}

func TestUUIDMoveHandlerTargetsIdentifierRoute(t *testing.T) {
	t.Parallel()

	files := fstest.MapFS{"people.cue": &fstest.MapFile{Data: []byte(identifiedHTMXSource)}}
	handler, err := newTestHandler(files, mapFileSystem{files: files})
	if err != nil {
		t.Fatal(err)
	}
	values := url.Values{
		"file":      {"people.cue"},
		"from_uuid": {"target"},
		"to":        {"0"},
	}
	response := servePOST(t, handler, "/move/uuid", values)
	if response.Code != http.StatusOK {
		t.Fatalf("UUID move status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}
	book, err := cuebook.New(files["people.cue"].Data)
	if err != nil {
		t.Fatal(err)
	}
	for index, want := range []string{"Target", "First"} {
		entry, err := book.GetEntryByIndex(index)
		if err != nil {
			t.Fatal(err)
		}
		if entry.Title != want {
			t.Errorf("entry %d title = %q, want %q", index, entry.Title, want)
		}
	}
}

func serveGET(t *testing.T, handler http.Handler, target string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "http://example.test"+target, nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func servePOST(t *testing.T, handler http.Handler, target string, values url.Values) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "http://example.test"+target, strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Origin", "http://example.test")
	request.Header.Set("HX-Request", "true")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
