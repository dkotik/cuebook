package htmx

import (
	"encoding/json"
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/dkotik/cuebook"
)

func TestEntryCopyRoundTripsThroughPaste(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		source   string
		index    int
		readOnly bool
		htmx     bool
		prefix   string
	}{
		{name: "writable page", source: `[{Name: "Ada", Active: true, Count: 42}]`},
		{name: "read only page", source: `[{Name: "Ada"}]`, readOnly: true},
		{name: "HTMX entry fragment", source: `[{Name: "Ada"}]`, htmx: true},
		{name: "mounted page", source: `[{Name: "Ada"}]`, prefix: "/books"},
		{name: "selected entry only", source: `[{Name: "First"}, {Name: "Second", Extra: "selected"}]`, index: 1},
		{name: "resolved definitions defaults references and details", source: `#person: {
	Name: string @cuebook(title)
	Note?: string @cuebook(detail)
	Missing?: string
	Nick: *"default" | string
	ID: Name
}
[...#person] & [{Name: "Ada", Note: "Details"}]`},
		{name: "HTML and unicode are safely encoded", source: `[{Name: "Café & friends", Note: "</pre><script>alert(1)</script>", Lines: "first\nsecond\t\"quoted\""}]`},
		{name: "nested values and lists", source: `[{Name: "Nested", Data: {Enabled: false, Items: [1, 2]}, Empty: null}]`},
	}
	payloadPattern := regexp.MustCompile(`<pre id="entry-cue-[0-9]+" hidden>([\s\S]*?)</pre>`)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			book, err := cuebook.New([]byte(tt.source))
			if err != nil {
				t.Fatal(err)
			}
			original, err := book.GetValue(tt.index)
			if err != nil {
				t.Fatal(err)
			}
			entryRange, err := cuebook.NewByteRange(original)
			if err != nil {
				t.Fatal(err)
			}
			files := fstest.MapFS{"people.cue": &fstest.MapFile{Data: []byte(tt.source)}}
			var handler http.Handler
			if tt.readOnly {
				handler, err = New(files, WithServeMuxPrefix(tt.prefix))
			} else {
				handler, err = NewWithCommitter(files, mapFileCommitter{files: files}, WithServeMuxPrefix(tt.prefix))
			}
			if err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest(http.MethodGet, "http://example.test"+tt.prefix+entryURL("people.cue", entryRange), nil)
			if tt.htmx {
				request.Header.Set("HX-Request", "true")
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				t.Fatalf("entry status = %d; body: %s", response.Code, response.Body.String())
			}
			body := response.Body.String()
			if !strings.Contains(body, `data-entry-copy="entry-cue-`+strconv.Itoa(tt.index)+`"`) || !strings.Contains(body, ">Copy CUE</button>") {
				t.Fatal("copy button missing or targets the wrong entry")
			}
			if !tt.htmx && !strings.Contains(body, `src="`+tt.prefix+`/assets/entry-copy.js"`) {
				t.Fatal("copy controller is not loaded")
			}
			matches := payloadPattern.FindAllStringSubmatch(body, -1)
			if len(matches) != 1 {
				t.Fatalf("copy payload count = %d, want 1", len(matches))
			}
			payload := html.UnescapeString(matches[0][1])
			copied := cuecontext.New().CompileString(payload)
			if err := copied.Validate(cue.Concrete(true)); err != nil {
				t.Fatalf("copied CUE is not standalone concrete data: %v; source: %s", err, payload)
			}
			if copied.Kind() != cue.StructKind {
				t.Fatalf("copied kind = %s, want struct", copied.Kind())
			}
			assertCopiedEntryEqual(t, copied, original)

			destination := fstest.MapFS{"people.cue": &fstest.MapFile{Data: []byte(tt.source)}}
			writable, err := NewWithCommitter(destination, mapFileCommitter{files: destination}, WithServeMuxPrefix(tt.prefix))
			if err != nil {
				t.Fatal(err)
			}
			pasted := submitPaste(writable, tt.prefix+"/paste", url.Values{"file": {"people.cue"}, "source": {payload}})
			if pasted.Code != http.StatusOK {
				t.Fatalf("paste status = %d; flash: %s", pasted.Code, pasted.Header().Get("HX-Trigger"))
			}
			updated, err := cuebook.New(destination["people.cue"].Data)
			if err != nil {
				t.Fatal(err)
			}
			before, _ := book.Len()
			after, _ := updated.Len()
			if after != before+1 {
				t.Fatalf("entry count = %d, want %d", after, before+1)
			}
			newEntry, err := updated.GetValue(after - 1)
			if err != nil {
				t.Fatal(err)
			}
			assertCopiedEntryEqual(t, newEntry, original)

			listRequest := httptest.NewRequest(http.MethodGet, "http://example.test"+tt.prefix+"/?file=people.cue", nil)
			listResponse := httptest.NewRecorder()
			handler.ServeHTTP(listResponse, listRequest)
			if strings.Contains(listResponse.Body.String(), "data-entry-copy=") || payloadPattern.MatchString(listResponse.Body.String()) {
				t.Error("list view unexpectedly exports copy payloads")
			}
		})
	}
}

func assertCopiedEntryEqual(t *testing.T, got, want cue.Value) {
	t.Helper()
	decode := func(value cue.Value) any {
		data, err := value.MarshalJSON()
		if err != nil {
			t.Fatal(err)
		}
		var decoded any
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatal(err)
		}
		return decoded
	}
	gotData, wantData := decode(got), decode(want)
	if !reflect.DeepEqual(gotData, wantData) {
		t.Errorf("copied data = %#v, want %#v", gotData, wantData)
	}
}
