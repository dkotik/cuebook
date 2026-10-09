package htmx

import (
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/dkotik/cuebook"
)

func TestEntryMoveDialog(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		prefix   string
		file     string
		htmx     bool
		readOnly bool
		onlyFile bool
	}{
		{name: "writable page"},
		{name: "HTMX entry fragment", htmx: true},
		{name: "mounted entry page", prefix: "/books"},
		{name: "archived entry can move", file: "archive/source.cue"},
		{name: "current file is the only file", onlyFile: true},
		{name: "read only has no move action", readOnly: true},
	}
	optionPattern := regexp.MustCompile(`<label class="radio is-block mb-3([^"]*)">\s*<input type="radio" name="destination" value="([^"]+)"([^>]*)>`)
	dialogPattern := regexp.MustCompile(`(?s)<dialog id="entry-move-dialog".*?</dialog>`)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			file := tt.file
			if file == "" {
				file = "source.cue"
			}
			const source = `[{Name: "Keep me"}, {Name: "Move me"}]`
			const destination = "nested/destination & friends.cue"
			files := fstest.MapFS{
				file:        &fstest.MapFile{Data: []byte(source)},
				"notes.txt": &fstest.MapFile{Data: []byte("not CUE")},
			}
			if !tt.onlyFile {
				files[destination] = &fstest.MapFile{Data: []byte(`[{Name: "Existing"}]`)}
				files["archive/other.cue"] = &fstest.MapFile{Data: []byte(`[]`)}
			}
			book, err := cuebook.New([]byte(source))
			if err != nil {
				t.Fatal(err)
			}
			value, err := book.GetValue(1)
			if err != nil {
				t.Fatal(err)
			}
			entryRange, err := cuebook.NewByteRange(value)
			if err != nil {
				t.Fatal(err)
			}
			var handler http.Handler
			if tt.readOnly {
				handler, err = New(files, WithServeMuxPrefix(tt.prefix))
			} else {
				handler, err = NewWithCommitter(files, mapFileCommitter{files: files}, WithServeMuxPrefix(tt.prefix))
			}
			if err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest(http.MethodGet, "http://example.test"+tt.prefix+entryURL(file, entryRange), nil)
			if tt.htmx {
				request.Header.Set("HX-Request", "true")
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				t.Fatalf("entry status = %d; body: %s", response.Code, response.Body.String())
			}
			body := response.Body.String()
			if tt.readOnly {
				if strings.Contains(body, "data-entry-move=") || dialogPattern.MatchString(body) {
					t.Fatal("read-only entry exposes the move action")
				}
				return
			}
			actionStart := strings.Index(body, `<div class="buttons entry-actions">`)
			if actionStart < 0 {
				t.Fatal("bottom action row is missing")
			}
			actionEnd := strings.Index(body[actionStart:], "</div>")
			if actionEnd < 0 || !strings.Contains(body[actionStart:actionStart+actionEnd], `data-entry-move="entry-move-dialog"`) {
				t.Fatal("move button is not in the bottom action row")
			}
			dialog := dialogPattern.FindString(body)
			for _, want := range []string{
				`action="` + tt.prefix + `/move"`,
				`hx-post="` + tt.prefix + `/move"`,
				`hx-target="#workspace" hx-swap="outerHTML"`,
				`name="file" value="` + html.EscapeString(file) + `"`,
				`name="from" value="1"`,
				`data-entry-move-dismiss autofocus>Cancel`,
				`type="submit" data-entry-move-confirm disabled>Confirm move`,
			} {
				if !strings.Contains(dialog, want) {
					t.Errorf("dialog missing %q", want)
				}
			}
			if strings.Contains(dialog, "hx-confirm") || strings.Contains(dialog, "notes.txt") {
				t.Error("dialog adds an extra confirmation or lists a non-CUE file")
			}
			options := optionPattern.FindAllStringSubmatch(dialog, -1)
			wantOptions := 3
			if tt.onlyFile {
				wantOptions = 1
			}
			if len(options) != wantOptions {
				t.Fatalf("file options = %d, want %d", len(options), wantOptions)
			}
			currentFound := false
			for _, option := range options {
				name := html.UnescapeString(option[2])
				if name == file {
					currentFound = true
					if !strings.Contains(option[1], "has-text-danger") || !strings.Contains(option[3], "disabled") {
						t.Error("current file is not red and disabled")
					}
				} else if strings.Contains(option[1], "has-text-danger") || strings.Contains(option[3], "disabled") || !strings.Contains(option[3], "required") {
					t.Errorf("destination %q is not selectable", name)
				}
			}
			if !currentFound {
				t.Fatal("current file is not listed")
			}
			if !tt.htmx && !strings.Contains(body, `src="`+tt.prefix+`/assets/entry-move-dialog.js"`) {
				t.Error("move dialog controller is not loaded")
			}
			if tt.onlyFile {
				return
			}

			values := url.Values{"file": {file}, "from": {"1"}, "destination": {destination}}
			moveRequest := httptest.NewRequest(http.MethodPost, "http://example.test"+tt.prefix+"/move", strings.NewReader(values.Encode()))
			moveRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			moveRequest.Header.Set("Origin", "http://example.test")
			moveRequest.Header.Set("HX-Request", "true")
			moved := httptest.NewRecorder()
			handler.ServeHTTP(moved, moveRequest)
			if moved.Code != http.StatusOK {
				t.Fatalf("confirm move status = %d; body: %s", moved.Code, moved.Body.String())
			}
			assertEntryTitles(t, files[file].Data, []string{"Keep me"})
			assertEntryTitles(t, files[destination].Data, []string{"Existing", "Move me"})
			if !strings.Contains(moved.Body.String(), `<main id="workspace"`) || strings.Contains(moved.Body.String(), "data-entry-move=") {
				t.Error("move did not return the destination list workspace")
			}
		})
	}
}
