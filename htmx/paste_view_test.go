package htmx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/dkotik/cuebook"
)

func TestPasteAvailableOnlyInWritableListView(t *testing.T) {
	t.Parallel()
	source := []byte(`[{Name: "Existing"}]`)
	document, err := cuebook.New(source)
	if err != nil {
		t.Fatal(err)
	}
	value, err := document.GetValue(0)
	if err != nil {
		t.Fatal(err)
	}
	entryRange, err := cuebook.NewByteRange(value)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name      string
		route     string
		prefix    string
		readOnly  bool
		htmx      bool
		wantPaste bool
	}{
		{name: "writable list page", route: "/?file=contacts.cue", wantPaste: true},
		{name: "writable list fragment", route: "/?file=contacts.cue", htmx: true, wantPaste: true},
		{name: "mounted list", route: "/?file=contacts.cue", prefix: "/books", wantPaste: true},
		{name: "read only", route: "/?file=contacts.cue", readOnly: true},
		{name: "no file selected", route: "/"},
		{name: "entry page", route: entryURL("contacts.cue", entryRange)},
		{name: "entry fragment", route: entryURL("contacts.cue", entryRange), htmx: true},
		{name: "search results", route: "/search?q=Existing", htmx: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			files := fstest.MapFS{"contacts.cue": &fstest.MapFile{Data: source}}
			var handler http.Handler
			var err error
			if tt.readOnly {
				handler, err = New(files, WithServeMuxPrefix(tt.prefix))
			} else {
				handler, err = NewWithCommitter(files, &recordingCommitter{}, WithServeMuxPrefix(tt.prefix))
			}
			if err != nil {
				t.Fatal(err)
			}
			if strings.HasPrefix(tt.route, "/search") {
				waitForSearchResponse(t, handler, "Existing")
			}
			request := httptest.NewRequest(http.MethodGet, "http://example.test"+tt.prefix+tt.route, nil)
			if tt.htmx {
				request.Header.Set("HX-Request", "true")
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d; body: %s", response.Code, response.Body.String())
			}
			body := response.Body.String()
			if got := strings.Contains(body, `data-paste-file="contacts.cue"`); got != tt.wantPaste {
				t.Errorf("paste enabled = %t, want %t", got, tt.wantPaste)
			}
			if tt.wantPaste && !strings.Contains(body, `data-paste-url="`+tt.prefix+`/paste"`) {
				t.Error("paste destination missing mounted route")
			}
			if !tt.htmx {
				for _, want := range []string{
					`src="` + tt.prefix + `/assets/entry-paste.js"`,
					`src="` + tt.prefix + `/assets/flash.js"`,
					`id="flash-message"`,
					`role="alert" hidden`,
				} {
					if !strings.Contains(body, want) {
						t.Errorf("page missing %q", want)
					}
				}
			}
		})
	}
}
