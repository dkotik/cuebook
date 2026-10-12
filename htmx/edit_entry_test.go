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

func TestEntryFieldSaveReplacesCurrentEntryURL(t *testing.T) {
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
		{name: "non-HTMX form renders updated entry", field: "Name", value: "Updated browser entry", browser: true},
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
			for _, want := range []string{
				`<div class="modal-background" data-field-edit-dismiss></div>`,
				`<form id="field-edit-modal-card" class="modal-card field-edit-modal-form"`,
				`name="view" value="entry"`,
				`data-field-edit-dismiss>Cancel</button>`,
			} {
				if !strings.Contains(form.Body.String(), want) {
					t.Errorf("entry edit dialog does not contain %q: %s", want, form.Body.String())
				}
			}
			if strings.Contains(form.Body.String(), `hx-target="closest .field-item"`) {
				t.Fatal("entry edit form unexpectedly targets the inline field")
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
				if response.Code != http.StatusOK || response.Header().Get("HX-Replace-Url") != wantURL {
					t.Fatalf("non-HTMX response: status %d, replacement %q; want %s", response.Code, response.Header().Get("HX-Replace-Url"), wantURL)
				}
				for _, header := range []string{"HX-Location", "HX-Redirect", "Location"} {
					if got := response.Header().Get(header); got != "" {
						t.Errorf("non-HTMX response unexpectedly sets %s: %q", header, got)
					}
				}
			} else {
				if response.Code != http.StatusOK {
					t.Fatalf("HTMX response status = %d; body: %s", response.Code, response.Body.String())
				}
				if got := response.Header().Get("HX-Replace-Url"); got != wantURL {
					t.Fatalf("HX-Replace-Url = %q; want %q", got, wantURL)
				}
				for _, header := range []string{"HX-Location", "HX-Redirect", "Location"} {
					if got := response.Header().Get(header); got != "" {
						t.Errorf("HTMX response unexpectedly sets %s: %q", header, got)
					}
				}
			}
			body := response.Body.String()
			for _, want := range []string{`<main id="workspace"`, `id="entry-1"`, "data-entry-copy=", "data-file-picker-open="} {
				if !strings.Contains(body, want) {
					t.Errorf("refreshed entry missing %q", want)
				}
			}
			if strings.Contains(body, "Unrelated entry") {
				t.Error("response contains another entry from the file list")
			}
			if !tt.browser && strings.Contains(body, "<!doctype html>") {
				t.Error("HTMX save returned a full page instead of a workspace fragment")
			}
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
