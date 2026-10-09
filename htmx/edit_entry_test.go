package htmx

import (
	"bytes"
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"

	"strings"
	"testing"
	"testing/fstest"

	"cuelang.org/go/cue"
	"github.com/dkotik/cuebook"
)

func TestEntryFieldSaveRefreshesCurrentEntry(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		field     string
		value     string
		prefix    string
		browser   bool
		unchanged bool
	}{
		{name: "longer title updates byte range", field: "Name", value: "A much longer updated title"},
		{name: "shorter title updates byte range", field: "Name", value: "A"},
		{name: "populating an optional detail", field: "Note", value: "New detail"},
		{name: "blank secret preserves entry view", field: "Password", unchanged: true},
		{name: "mounted entry view", field: "Name", value: "Updated mounted entry", prefix: "/catalog"},
		{name: "browser form returns to same entry", field: "Name", value: "Updated browser entry", browser: true},
	}
	const source = `#person: {
	Name: string @cuebook(title)
	Email: string & =~"^[^@]+@[^@]+$"
	Note?: string @cuebook(detail)
	Password?: string @cuebook(argon2id)
}
[...#person] & [
	{Name: "Unrelated entry", Email: "other@example.test"},
	{Name: "Ada", Email: "ada@example.test", Password: "stored-secret"},
]`
	editLinks := regexp.MustCompile(`hx-get="([^"]*/edit\?[^"]*)"`)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			files := fstest.MapFS{"people.cue": &fstest.MapFile{Data: []byte(source)}}
			handler, err := NewWithCommitter(files, mapFileCommitter{files: files}, WithServeMuxPrefix(tt.prefix))
			if err != nil {
				t.Fatal(err)
			}
			book, err := cuebook.New(files["people.cue"].Data)
			if err != nil {
				t.Fatal(err)
			}
			original, err := book.GetValue(1)
			if err != nil {
				t.Fatal(err)
			}
			originalRange, err := cuebook.NewByteRange(original)
			if err != nil {
				t.Fatal(err)
			}
			originalURL := tt.prefix + entryURL("people.cue", originalRange)
			get := func(target string) *httptest.ResponseRecorder {
				request := httptest.NewRequest(http.MethodGet, "http://example.test"+target, nil)
				request.Header.Set("HX-Request", "true")
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, request)
				if response.Code != http.StatusOK {
					t.Fatalf("GET %s: status %d; %s", target, response.Code, response.Body.String())
				}
				return response
			}
			page := get(originalURL)
			var formURL *url.URL
			for _, match := range editLinks.FindAllStringSubmatch(page.Body.String(), -1) {
				link, err := url.Parse(html.UnescapeString(match[1]))
				if err != nil {
					t.Fatal(err)
				}
				if link.Query().Get("field") == tt.field {
					formURL = link
					break
				}
			}
			if formURL == nil || formURL.Query().Get("view") != "entry" {
				t.Fatal("edit link loses the entry-view context")
			}
			form := get(formURL.String())
			if !strings.Contains(form.Body.String(), `name="view" value="entry"`) {
				t.Fatal("edit form loses the entry-view context")
			}
			cancelQuery := formURL.Query()
			cancelQuery.Set("mode", "view")
			cancelURL := formURL.Path + "?" + cancelQuery.Encode()
			if !strings.Contains(form.Body.String(), html.EscapeString(cancelURL)) {
				t.Fatal("cancel link loses the entry-view context")
			}
			if !strings.Contains(get(cancelURL).Body.String(), "view=entry") {
				t.Fatal("cancelled field loses the entry-view context")
			}

			values := editValues("people.cue", "1", tt.field, tt.value)
			values.Set("view", "entry")
			request := httptest.NewRequest(http.MethodPost, "http://example.test"+tt.prefix+"/edit", strings.NewReader(values.Encode()))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			request.Header.Set("Origin", "http://example.test")
			if !tt.browser {
				request.Header.Set("HX-Request", "true")
				request.Header.Set("HX-Current-URL", "http://example.test"+originalURL)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if got := response.Header().Get("HX-Redirect"); got != "" {
				t.Fatalf("unexpected redirect: %s", got)
			}
			updated, err := cuebook.New(files["people.cue"].Data)
			if err != nil {
				t.Fatal(err)
			}
			value, err := updated.GetValue(1)
			if err != nil {
				t.Fatal(err)
			}
			newRange, err := cuebook.NewByteRange(value)
			if err != nil {
				t.Fatal(err)
			}
			wantURL := tt.prefix + entryURL("people.cue", newRange)
			if !tt.unchanged && wantURL == originalURL {
				t.Fatal("test edit did not change the entry byte range")
			}
			if tt.unchanged && !bytes.Equal(files["people.cue"].Data, []byte(source)) {
				t.Fatal("blank secret changed the document")
			}
			if !tt.unchanged {
				field, err := value.LookupPath(cue.ParsePath(tt.field)).String()
				if err != nil || field != tt.value {
					t.Fatalf("saved field = %q, err = %v; want %q", field, err, tt.value)
				}
			}
			if tt.browser {
				if response.Code != http.StatusSeeOther || response.Header().Get("Location") != wantURL {
					t.Fatalf("browser response: status %d, location %q; want %s", response.Code, response.Header().Get("Location"), wantURL)
				}
				follow := httptest.NewRequest(http.MethodGet, "http://example.test"+wantURL, nil)
				response = httptest.NewRecorder()
				handler.ServeHTTP(response, follow)
				if response.Code != http.StatusOK {
					t.Fatalf("entry reload status = %d", response.Code)
				}
			} else {
				if response.Code != http.StatusOK || response.Header().Get("HX-Replace-Url") != wantURL {
					t.Fatalf("HTMX response: status %d, replacement %q; want %s", response.Code, response.Header().Get("HX-Replace-Url"), wantURL)
				}
			}
			body := response.Body.String()
			for _, want := range []string{`<main id="workspace"`, `id="entry-1"`, "data-entry-copy=", "data-entry-move="} {
				if !strings.Contains(body, want) {
					t.Errorf("refreshed entry missing %q", want)
				}
			}
			for _, unwanted := range []string{"Unrelated entry", `id="add-entry-heading"`} {
				if strings.Contains(body, unwanted) {
					t.Errorf("response contains list/full-page content %q", unwanted)
				}
			}
			if !tt.browser && strings.Contains(body, "<!doctype html>") {
				t.Error("HTMX save returned a full page instead of a workspace fragment")
			}
			get(wantURL)
		})
	}
}

func TestEntryFieldValidationErrorDoesNotNavigate(t *testing.T) {
	t.Parallel()
	files := fstest.MapFS{"people.cue": &fstest.MapFile{Data: []byte("#entry: {Name: string, Count: int}\n[...#entry] & [{Name: \"Ada\", Count: 1}]")}}
	handler, err := NewWithCommitter(files, mapFileCommitter{files: files})
	if err != nil {
		t.Fatal(err)
	}
	original := bytes.Clone(files["people.cue"].Data)
	values := editValues("people.cue", "0", "Count", "not a number")
	values.Set("view", "entry")
	response := submitEdit(t, handler, true, values)
	if response.Code != http.StatusNoContent || response.Body.Len() != 0 {
		t.Fatalf("status %d, body %q; want empty 204", response.Code, response.Body.String())
	}
	for _, header := range []string{"HX-Redirect", "HX-Replace-Url", "Location"} {
		if response.Header().Get(header) != "" {
			t.Errorf("validation error sets %s", header)
		}
	}
	if response.Header().Get("HX-Trigger") == "" {
		t.Fatal("validation error has no flash message")
	}
	if !bytes.Equal(files["people.cue"].Data, original) {
		t.Fatal("validation error modified the document")
	}
}
