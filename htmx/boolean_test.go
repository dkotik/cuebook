package htmx

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"cuelang.org/go/cue"
	"github.com/dkotik/cuebook"
)

func TestBooleanFormRendering(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		declaration string
		existing    string
		edit        bool
		optional    bool
		secret      bool
		selected    string
	}{
		{name: "required add starts unset", declaration: "Enabled: bool", existing: "true", selected: "not_set"},
		{name: "required add ignores true default", declaration: "Enabled: *true | bool", existing: "true", selected: "not_set"},
		{name: "optional add starts unset", declaration: "Enabled?: bool", optional: true, selected: "not_set"},
		{name: "optional populated add starts unset", declaration: "Enabled?: bool", existing: "true", optional: true, selected: "not_set"},
		{name: "optional add ignores true default", declaration: "Enabled?: *true | bool", optional: true, selected: "not_set"},
		{name: "edit selects false", declaration: "Enabled: bool", existing: "false", edit: true, selected: "false"},
		{name: "edit selects true", declaration: "Enabled: bool", existing: "true", edit: true, selected: "true"},
		{name: "optional edit selects false", declaration: "Enabled?: bool", existing: "false", edit: true, optional: true, selected: "false"},
		{name: "optional edit selects true", declaration: "Enabled?: bool", existing: "true", edit: true, optional: true, selected: "true"},
		{name: "optional edit stays unset", declaration: "Enabled?: bool", edit: true, optional: true, selected: "not_set"},
		{name: "optional edit ignores true default", declaration: "Enabled?: *true | bool", edit: true, optional: true, selected: "not_set"},
		{name: "secret required add stays password", declaration: "Enabled: bool @cuebook(argon2id)", existing: "true", secret: true},
		{name: "secret optional add stays password", declaration: "Enabled?: bool @cuebook(argon2id)", optional: true, secret: true},
		{name: "secret false edit stays password", declaration: "Enabled: bool @cuebook(argon2id)", existing: "false", edit: true, secret: true},
		{name: "secret true edit stays password", declaration: "Enabled: bool @cuebook(argon2id)", existing: "true", edit: true, secret: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			entry := `Name: "Existing"`
			if tt.existing != "" {
				entry += ", Enabled: " + tt.existing
			}
			source := "#entry: {\nName: string @cuebook(title)\n" + tt.declaration + "\n}\n[...#entry] & [{" + entry + "}]"
			files := fstest.MapFS{"booleans.cue": &fstest.MapFile{Data: []byte(source)}}
			handler, err := NewWithCommitter(files, &recordingCommitter{})
			if err != nil {
				t.Fatal(err)
			}

			target := "http://example.test/?file=booleans.cue"
			inputName := "entry.1.value"
			if tt.edit {
				query := url.Values{"file": {"booleans.cue"}, "entry": {"0"}, "field": {"Enabled"}}
				target = "http://example.test/edit?" + query.Encode()
				inputName = "value"
			}
			request := httptest.NewRequest(http.MethodGet, target, nil)
			request.Header.Set("HX-Request", "true")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
			}
			form := response.Body.String()
			if !tt.edit {
				start := strings.Index(form, `<form id="add-entry-form"`)
				if start < 0 {
					t.Fatalf("add form not found: %s", form)
				}
				form = form[start:]
				end := strings.Index(form, "</form>")
				if end < 0 {
					t.Fatalf("add form is not closed: %s", form)
				}
				form = form[:end]
				if !strings.Contains(form, `name="entry.1.field" value="Enabled"`) {
					t.Errorf("boolean field name missing: %s", form)
				}
			}

			if tt.secret {
				want := `type="password" name="` + inputName + `" value=""`
				if !strings.Contains(form, want) {
					t.Errorf("secret boolean must render an empty password input containing %q: %s", want, form)
				}
				for _, unwanted := range []string{`type="radio"`, `role="radiogroup"`, `boolean-toggle`} {
					if strings.Contains(form, unwanted) {
						t.Errorf("secret boolean unexpectedly contains %q: %s", unwanted, form)
					}
				}
				return
			}

			components := regexp.MustCompile(`(?s)<boolean-toggle\b[^>]*>.*?</boolean-toggle>`).FindAllString(form, -1)
			if len(components) != 1 {
				t.Fatalf("boolean component count = %d, want 1: %s", len(components), form)
			}
			component := components[0]
			if strings.Count(component, "<button ") != 1 || strings.Count(component, "<input ") != 1 {
				t.Errorf("want one button and one hidden input per component: %s", component)
			}
			class, pressed, symbol := "button boolean-toggle", tt.selected, "✖"
			switch tt.selected {
			case "true":
				class, symbol = class+" has-text-info", "✔"
			case "false":
				class += " has-text-danger"
			case "not_set":
				pressed, symbol = "mixed", "☐"
			}
			toggleTag := `<boolean-toggle data-optional="` + strconv.FormatBool(tt.optional) + `"`
			if !tt.edit {
				toggleTag += ` data-allow-unset="true"`
			}
			toggleTag += ` expanded="true"`
			for _, want := range []string{
				toggleTag + `>`,
				`<input type="hidden" name="` + inputName + `" value="` + tt.selected + `">`,
				`<button class="` + class + `" type="button" aria-label="Enabled" aria-pressed="` + pressed + `" title="Enabled: ` + tt.selected + `">` + symbol + `</button>`,
			} {
				if !strings.Contains(component, want) {
					t.Errorf("boolean component missing %q: %s", want, component)
				}
			}
			for _, unwanted := range []string{`type="radio"`, `<select`, `type="text" name="` + inputName + `"`} {
				if strings.Contains(form, unwanted) {
					t.Errorf("boolean form unexpectedly contains %q: %s", unwanted, form)
				}
			}
		})
	}
}

func TestBooleanEditSubmission(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		existing string
		value    string
		want     bool
	}{
		{name: "true saves as false", existing: "true", value: "false", want: false},
		{name: "false saves as true", existing: "false", value: "true", want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			source := "#entry: {Name: string @cuebook(title), Enabled: bool}\n[...#entry] & [{Name: \"Existing\", Enabled: " + tt.existing + "}]"
			files := fstest.MapFS{"booleans.cue": &fstest.MapFile{Data: []byte(source)}}
			handler, err := NewWithCommitter(files, mapFileCommitter{files: files})
			if err != nil {
				t.Fatal(err)
			}
			response := submitEdit(t, handler, true, editValues("booleans.cue", "0", "Enabled", tt.value))
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
			}
			document, err := cuebook.New(files["booleans.cue"].Data)
			if err != nil {
				t.Fatalf("saved source is invalid: %v", err)
			}
			entry, err := document.GetValue(0)
			if err != nil {
				t.Fatal(err)
			}
			if got, err := entry.LookupPath(cue.ParsePath("Enabled")).Bool(); err != nil || got != tt.want {
				t.Fatalf("saved Enabled = %t, err = %v; want %t", got, err, tt.want)
			}
		})
	}
}

func TestBooleanAddSubmission(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		declaration string
		value       string
		unset       bool
		want        bool
	}{
		{name: "required false overrides true default", declaration: "Enabled: *true | bool", value: "false"},
		{name: "required true saves", declaration: "Enabled: bool", value: "true", want: true},
		{name: "optional false saves", declaration: "Enabled?: bool", value: "false"},
		{name: "optional true saves", declaration: "Enabled?: bool", value: "true", want: true},
		{name: "optional unset is omitted", declaration: "Enabled?: bool", value: "not_set", unset: true},
		{name: "optional unset does not apply true default", declaration: "Enabled?: *true | bool", value: "not_set", unset: true},
		{name: "legacy optional empty input is omitted", declaration: "Enabled?: bool", unset: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			source := "#entry: {\nName: string @cuebook(title)\n" + tt.declaration + "\n}\n[...#entry] & []"
			files := fstest.MapFS{"booleans.cue": &fstest.MapFile{Data: []byte(source)}}
			handler, err := NewWithCommitter(files, mapFileCommitter{files: files})
			if err != nil {
				t.Fatal(err)
			}
			response := submitAdd(t, handler, true, addEntryValues("booleans.cue", [][2]string{
				{"Name", "Added"}, {"Enabled", tt.value},
			}))
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
			}
			document, err := cuebook.New(files["booleans.cue"].Data)
			if err != nil {
				t.Fatalf("saved source is invalid: %v", err)
			}
			if length, err := document.Len(); err != nil || length != 1 {
				t.Fatalf("entry count = %d, err = %v; want 1", length, err)
			}
			entry, err := document.GetValue(0)
			if err != nil {
				t.Fatal(err)
			}
			if got, err := entry.LookupPath(cue.ParsePath("Name")).String(); err != nil || got != "Added" {
				t.Fatalf("saved Name = %q, err = %v; want Added", got, err)
			}
			enabled := entry.LookupPath(cue.ParsePath("Enabled"))
			if tt.unset {
				if enabled.Exists() {
					t.Fatalf("unset optional Enabled was saved: %s", files["booleans.cue"].Data)
				}
				return
			}
			if got, err := enabled.Bool(); err != nil || got != tt.want {
				t.Fatalf("saved Enabled = %t, err = %v; want %t", got, err, tt.want)
			}
		})
	}
}
