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

	const filePath = "tasks.cue"
	source := []byte(`#entry: {
	Name: string @cuebook(title)
	Notes: string @cuebook(multiline)
}
[...#entry] & [{
	Name: "Tasks"
	Notes: """
		- [ ] Pending task
		- [x] Completed task
		"""
}]`)
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

	handler, err := New(fstest.MapFS{
		filePath: &fstest.MapFile{Data: source},
	})
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "http://example.test"+entryURL(filePath, entryRange), nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}

	body := response.Body.String()
	for _, checkbox := range []string{
		`<input disabled="" type="checkbox">`,
		`<input checked="" disabled="" type="checkbox">`,
	} {
		if !strings.Contains(body, checkbox) {
			t.Errorf("entry view does not contain task checkbox %q: %s", checkbox, body)
		}
	}
}
