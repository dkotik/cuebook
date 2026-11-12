package htmx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/dkotik/cuebook"
)

func TestEntryViewRendersTaskListCheckboxes(t *testing.T) {
	t.Parallel()

	const (
		filePath = "tasks.cue"
		taskList = "- [ ] Pending task\n- [x] Completed task"
	)
	checkboxes := []string{
		`<input disabled="" type="checkbox">`,
		`<input checked="" disabled="" type="checkbox">`,
	}
	newSource := func(attribute, notes string) []byte {
		return []byte(`#entry: {
	Name: string @cuebook(title)
	Notes: string ` + attribute + `
}
[...#entry] & [{
	Name: "Tasks"
	Notes: """
		` + strings.ReplaceAll(notes, "\n", "\n\t\t") + `
		"""
}]`)
	}

	tests := []struct {
		name           string
		source         []byte
		readOnly       bool
		fragment       bool
		edit           string // when set, POST this Notes value with view=entry and inspect the response
		wantCheckboxes bool
	}{
		{name: "read-only full page", source: newSource("@cuebook(multiline)", taskList), readOnly: true, wantCheckboxes: true},
		{name: "writable full page", source: newSource("@cuebook(multiline)", taskList), wantCheckboxes: true},
		{name: "writable HTMX fragment", source: newSource("@cuebook(multiline)", taskList), fragment: true, wantCheckboxes: true},
		{name: "entry view after editing notes into a task list", source: newSource("@cuebook(multiline)", "Plain notes"), fragment: true, edit: taskList, wantCheckboxes: true},
		{name: "single-line field is not rendered as Markdown", source: newSource("", taskList)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			files := fstest.MapFS{filePath: &fstest.MapFile{Data: tt.source}}
			var (
				handler http.Handler
				err     error
			)
			if tt.readOnly {
				handler, err = New(files)
			} else {
				handler, err = newTestHandler(files, mapFileSystem{files: files})
			}
			if err != nil {
				t.Fatal(err)
			}

			var request *http.Request
			if tt.edit == "" {
				request = httptest.NewRequest(http.MethodGet, "http://example.test"+firstEntryURL(t, tt.source, filePath), nil)
			} else {
				values := editValues(filePath, "0", "Notes", tt.edit)
				values.Set("view", "entry")
				request = httptest.NewRequest(http.MethodPost, "http://example.test/edit", strings.NewReader(values.Encode()))
				request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				request.Header.Set("Origin", "http://example.test")
				request.Header.Set("HX-Current-URL", "http://example.test"+firstEntryURL(t, tt.source, filePath))
			}
			if tt.fragment {
				request.Header.Set("HX-Request", "true")
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
			}

			body := response.Body.String()
			if got := strings.Contains(body, `<div class="field-markdown content">`); got != tt.wantCheckboxes {
				t.Errorf("Markdown container present = %t, want %t: %s", got, tt.wantCheckboxes, body)
			}
			for _, checkbox := range checkboxes {
				if got := strings.Contains(body, checkbox); got != tt.wantCheckboxes {
					t.Errorf("task checkbox %q present = %t, want %t: %s", checkbox, got, tt.wantCheckboxes, body)
				}
			}
		})
	}
}

func firstEntryURL(t *testing.T, source []byte, filePath string) string {
	t.Helper()
	book, err := cuebook.New(source)
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
	return entryURL(filePath, entryRange)
}
