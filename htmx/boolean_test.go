package htmx

import (
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
		{name: "required add starts false", declaration: "Enabled: bool", existing: "true", selected: "false"},
		{name: "required add ignores true default", declaration: "Enabled: *true | bool", existing: "true", selected: "false"},
		{name: "optional add starts unset", declaration: "Enabled?: bool", optional: true},
		{name: "optional add ignores true default", declaration: "Enabled?: *true | bool", optional: true},
		{name: "edit selects false", declaration: "Enabled: bool", existing: "false", edit: true, selected: "false"},
		{name: "edit selects true", declaration: "Enabled: bool", existing: "true", edit: true, selected: "true"},
		{name: "optional edit selects false", declaration: "Enabled?: bool", existing: "false", edit: true, selected: "false"},
		{name: "optional edit selects true", declaration: "Enabled?: bool", existing: "true", edit: true, selected: "true"},
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

			groups := regexp.MustCompile(`(?s)<div class="buttons has-addons boolean-toggle" role="radiogroup"[^>]*>(.*?)</div>`).FindAllStringSubmatch(form, -1)
			if len(groups) != 1 {
				t.Fatalf("boolean radio group count = %d, want 1: %s", len(groups), form)
			}
			group := groups[0][1]
			labels := regexp.MustCompile(`(?s)<label class="button(?: [^"]*)?"[^>]*>.*?</label>`).FindAllString(group, -1)
			choices := []string{"false", "true"}
			if tt.optional {
				choices = append([]string{""}, choices...)
				if !strings.Contains(group, "Not set") {
					t.Errorf("optional boolean has no Not set label: %s", group)
				}
			} else if strings.Contains(group, `value=""`) || strings.Contains(group, "Not set") {
				t.Errorf("required boolean unexpectedly offers an unset choice: %s", group)
			}
			if len(labels) != len(choices) || strings.Count(group, `type="radio"`) != len(choices) {
				t.Fatalf("want %d button labels containing radios, got %d labels: %s", len(choices), len(labels), group)
			}
			if got := strings.Count(group, " checked"); got != 1 {
				t.Errorf("checked radio count = %d, want 1: %s", got, group)
			}
			for _, choice := range choices {
				want := `<input class="is-sr-only" type="radio" name="` + inputName + `" value="` + choice + `"`
				if choice == tt.selected {
					want += " checked"
				}
				if !tt.optional {
					want += " required"
				}
				want += ">"
				found := false
				for _, label := range labels {
					if strings.Contains(label, want) {
						found = true
						if choice == "" && !strings.Contains(label, "Not set") {
							t.Errorf("unset radio is not inside its Not set label: %s", label)
						}
					}
				}
				if !found {
					t.Errorf("no button label contains %q: %s", want, group)
				}
			}
			if strings.Contains(form, `<select name="`+inputName+`"`) || strings.Contains(form, `type="text" name="`+inputName+`"`) {
				t.Errorf("boolean still renders a select or text input: %s", form)
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
		{name: "optional unset is omitted", declaration: "Enabled?: bool", unset: true},
		{name: "optional unset does not apply true default", declaration: "Enabled?: *true | bool", unset: true},
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
