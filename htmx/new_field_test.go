package htmx

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"testing/fstest"
)

func TestNewFieldAddsAFieldOfTheChosenKindAndValue(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		source string
		entry  string
		field  string
		kind   string
		value  string
		// others are submitted for the kinds that were not chosen, as the
		// form submits every value control.
		others string
		want   string
	}{
		{name: "string", source: `[{Name: "Ada"}]`, field: "Role", kind: "string", value: "dev", want: `[{Name: "Ada", Role: "dev"}]`},
		{name: "multiline string", source: `[{Name: "Ada"}]`, field: "Bio", kind: "multiline", value: "line one\r\nline two", want: "[{\n\t\tName: \"Ada\"\n\t\tBio: \"\"\"\n\t\t\tline one\n\t\t\tline two\n\t\t\t\"\"\" @cuebook(multiline)\n\t}]"},
		{name: "number", source: `[{Name: "Ada"}]`, field: "Age", kind: "number", value: "36", want: `[{Name: "Ada", Age: 36}]`},
		{name: "boolean", source: `[{Name: "Ada"}]`, field: "Active", kind: "boolean", value: "true", want: `[{Name: "Ada", Active: true}]`},
		{name: "empty string", source: `[{Name: "Ada"}]`, field: "Role", kind: "string", want: `[{Name: "Ada", Role: ""}]`},
		{name: "empty number is zero", source: `[{Name: "Ada"}]`, field: "Age", kind: "number", want: `[{Name: "Ada", Age: 0}]`},
		{name: "values of other kinds are ignored", source: `[{Name: "Ada"}]`, field: "Age", kind: "number", value: "7", others: "ignored", want: `[{Name: "Ada", Age: 7}]`},
		{name: "name is trimmed", source: `[{Name: "Ada"}]`, field: "  Role  ", kind: "string", value: "dev", want: `[{Name: "Ada", Role: "dev"}]`},
		{name: "selected entry only", source: `[{Name: "Ada"}, {Name: "Grace"}]`, entry: "1", field: "Role", kind: "string", value: "dev", want: `[{Name: "Ada"}, {Name: "Grace", Role: "dev"}]`},
		{name: "open schema", source: "#person: {Name: string, ...}\n[...#person] & [{Name: \"Ada\"}]", field: "Role", kind: "string", value: "dev", want: "#person: {Name: string, ...}\n[...#person] & [{Name: \"Ada\", Role: \"dev\"}]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			files := fstest.MapFS{"people.cue": &fstest.MapFile{Data: []byte(tt.source)}}
			handler, err := newTestHandler(files, mapFileSystem{files: files})
			if err != nil {
				t.Fatal(err)
			}
			entry := tt.entry
			if entry == "" {
				entry = "0"
			}
			values := newFieldValues("people.cue", entry, tt.field, tt.kind, tt.value)
			if tt.others != "" {
				for _, kind := range []string{"string", "multiline", "number", "boolean"} {
					if kind != tt.kind {
						values.Set("value_"+kind, tt.others)
					}
				}
			}
			response := submitNewField(handler, values)
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d; flash: %s; body: %s", response.Code, response.Header().Get("HX-Trigger"), response.Body.String())
			}
			if got := string(files["people.cue"].Data); got != tt.want {
				t.Fatalf("saved source =\n%s\nwant\n%s", got, tt.want)
			}
			body := response.Body.String()
			if !strings.Contains(body, strings.TrimSpace(tt.field)) {
				t.Errorf("response does not show the new field: %s", body)
			}
			// Adding a field moves the end of the entry, so its URL changes.
			if got := response.Header().Get("HX-Replace-Url"); !strings.HasPrefix(got, "/entry?") {
				t.Errorf("HX-Replace-Url = %q, want the updated entry URL", got)
			}
		})
	}
}

func TestNewFieldRejectsInvalidRequests(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		source   string
		values   url.Values
		readOnly bool
		want     string
	}{
		{name: "read only source", source: `[{Name: "Ada"}]`, values: newFieldValues("people.cue", "0", "Role", "string", "dev"), readOnly: true, want: "This source is read-only."},
		{name: "closed schema", source: "#person: {Name: string}\n[...#person] & [{Name: \"Ada\"}]", values: newFieldValues("people.cue", "0", "Role", "string", "dev"), want: "This entry does not accept new fields."},
		{name: "existing field", source: `[{Name: "Ada"}]`, values: newFieldValues("people.cue", "0", "Name", "string", "Grace"), want: "The field Name already exists."},
		{name: "blank name", source: `[{Name: "Ada"}]`, values: newFieldValues("people.cue", "0", "  ", "string", "dev"), want: "The field name is required."},
		{name: "unknown kind", source: `[{Name: "Ada"}]`, values: newFieldValues("people.cue", "0", "Role", "list", "dev"), want: "Choose a field type."},
		{name: "missing kind", source: `[{Name: "Ada"}]`, values: newFieldValues("people.cue", "0", "Role", "", "dev"), want: "Choose a field type."},
		{name: "value that is not a number", source: `[{Name: "Ada"}]`, values: newFieldValues("people.cue", "0", "Age", "number", "ten"), want: "The value of field Age is not a number."},
		{name: "value that is not a boolean", source: `[{Name: "Ada"}]`, values: newFieldValues("people.cue", "0", "Active", "boolean", "maybe"), want: "The value of field Active is invalid."},
		{name: "invalid entry index", source: `[{Name: "Ada"}]`, values: newFieldValues("people.cue", "first", "Role", "string", "dev"), want: "The entry index is invalid."},
		{name: "missing entry", source: `[{Name: "Ada"}]`, values: newFieldValues("people.cue", "3", "Role", "string", "dev"), want: "entry not found"},
		{name: "pattern constraint rejects kind", source: "#person: {Name: string, [string]: string}\n[...#person] & [{Name: \"Ada\"}]", values: newFieldValues("people.cue", "0", "Age", "number", "3"), want: "The new field does not satisfy the CUE constraints"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			files := fstest.MapFS{"people.cue": &fstest.MapFile{Data: []byte(tt.source)}}
			storage := &recordingFileSystem{}
			var handler http.Handler
			var err error
			if tt.readOnly {
				handler, err = New(files)
			} else {
				handler, err = newTestHandler(files, storage)
			}
			if err != nil {
				t.Fatal(err)
			}
			response := submitNewField(handler, tt.values)
			if response.Code == http.StatusOK {
				t.Fatalf("status = %d, want an error; body: %s", response.Code, response.Body.String())
			}
			if got := response.Header().Get("HX-Trigger") + response.Body.String(); !strings.Contains(got, tt.want) {
				t.Errorf("response does not explain %q: %s", tt.want, got)
			}
			if storage.calls != 0 {
				t.Errorf("write calls = %d, want 0", storage.calls)
			}
		})
	}
}

func TestNewFieldRejectsCrossOriginRequests(t *testing.T) {
	t.Parallel()
	files := fstest.MapFS{"people.cue": &fstest.MapFile{Data: []byte(`[{Name: "Ada"}]`)}}
	storage := &recordingFileSystem{}
	handler, err := newTestHandler(files, storage)
	if err != nil {
		t.Fatal(err)
	}
	request := newFieldRequestFor(newFieldValues("people.cue", "0", "Role", "string", "dev"))
	request.Header.Set("Origin", "http://attacker.test")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
	if storage.calls != 0 {
		t.Errorf("write calls = %d, want 0", storage.calls)
	}
}

// newFieldValues submits value in the control of the kind. An unknown kind has
// no control, so its value goes nowhere.
func newFieldValues(fileName, entry, name, kind, value string) url.Values {
	values := url.Values{"file": {fileName}, "entry": {entry}, "name": {name}, "kind": {kind}}
	switch kind {
	case "string", "multiline", "number", "boolean":
		values.Set("value_"+kind, value)
	}
	return values
}

func newFieldRequestFor(values url.Values) *http.Request {
	request := httptest.NewRequest(http.MethodPost, "http://example.test/add-field", strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Origin", "http://example.test")
	request.Header.Set("HX-Request", "true")
	return request
}

func submitNewField(handler http.Handler, values url.Values) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, newFieldRequestFor(values))
	return response
}
