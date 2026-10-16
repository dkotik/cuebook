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

func TestInlineBooleanRendering(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		declaration string
		existing    string
		prefix      string
		selected    string
		optional    bool
		readOnly    bool
		list        bool
		secret      bool
	}{
		{name: "stored false", declaration: "Enabled: bool", existing: "false", selected: "false"},
		{name: "stored true", declaration: "Enabled: bool", existing: "true", selected: "true"},
		{name: "optional false", declaration: "Enabled?: bool", existing: "false", selected: "false", optional: true},
		{name: "optional true", declaration: "Enabled?: bool", existing: "true", selected: "true", optional: true},
		{name: "optional unset", declaration: "Enabled?: bool", selected: "not_set", optional: true},
		{name: "unset optional ignores true default", declaration: "Enabled?: *true | bool", selected: "not_set", optional: true},
		{name: "unset optional detail", declaration: "Enabled?: bool @cuebook(detail)", selected: "not_set", optional: true},
		{name: "mounted routes", declaration: "Enabled: bool", existing: "true", prefix: "/catalog", selected: "true"},
		{name: "read only true", declaration: "Enabled: bool", existing: "true", selected: "true", readOnly: true},
		{name: "read only false", declaration: "Enabled: bool", existing: "false", selected: "false", readOnly: true},
		{name: "read only unset", declaration: "Enabled?: bool", selected: "not_set", readOnly: true},
		{name: "read only optional false", declaration: "Enabled?: bool", existing: "false", selected: "false", readOnly: true},
		{name: "read only optional true", declaration: "Enabled?: bool", existing: "true", selected: "true", readOnly: true},
		{name: "read only unset ignores true default", declaration: "Enabled?: *true | bool", selected: "not_set", readOnly: true},
		{name: "read only secret stays masked", declaration: "Enabled: bool @cuebook(argon2id)", existing: "true", readOnly: true, secret: true},
		{name: "secret stays modal", declaration: "Enabled: bool @cuebook(argon2id)", existing: "true", secret: true},
		{name: "list stays modal", declaration: "Enabled: bool", existing: "false", list: true},
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
			var handler http.Handler
			var err error
			if tt.readOnly {
				handler, err = New(files, WithServeMuxPrefix(tt.prefix))
			} else {
				handler, err = NewWithCommitter(files, mapFileCommitter{files: files}, WithServeMuxPrefix(tt.prefix))
			}
			if err != nil {
				t.Fatal(err)
			}
			target := tt.prefix + "/?file=booleans.cue"
			if !tt.list {
				target = inlineBooleanEntryURL(t, files, 0, tt.prefix)
			}
			request := httptest.NewRequest(http.MethodGet, "http://example.test"+target, nil)
			request.Header.Set("HX-Request", "true")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				t.Fatalf("GET status = %d; body: %s", response.Code, response.Body.String())
			}
			article := inlineBooleanArticle(t, response.Body.String())
			forms := regexp.MustCompile(`(?s)<form class="boolean-field-form"[^>]*>.*?</form>`).FindAllString(article, -1)
			if tt.readOnly || tt.list || tt.secret {
				if len(forms) != 0 || strings.Contains(article, "boolean-toggle") {
					t.Errorf("unexpected inline boolean controls: %s", article)
				}
				if tt.readOnly {
					output := `<output><span class="has-text-grey">☐</span></output>`
					switch tt.selected {
					case "true":
						output = `<output><span class="has-text-info">✔</span></output>`
					case "false":
						output = `<output><span class="has-text-danger">✖</span></output>`
					}
					if tt.secret {
						output = "<output>••••••••</output>"
					}
					if !strings.Contains(article, output) || strings.Contains(article, "data-field-edit-trigger") {
						t.Errorf("read-only field must remain static output: %s", article)
					}
				} else {
					if !strings.Contains(article, `aria-label="Edit Enabled value"`) || !strings.Contains(article, `hx-target="#field-edit-modal"`) {
						t.Errorf("boolean field lost its modal edit control: %s", article)
					}
					if tt.secret && (!strings.Contains(article, "<output>••••••••</output>") || strings.Contains(article, "<output>true</output>")) {
						t.Errorf("secret boolean value is not masked: %s", article)
					}
				}
			} else {
				if len(forms) != 1 {
					t.Fatalf("inline boolean form count = %d, want 1: %s", len(forms), article)
				}
				form := forms[0]
				for _, want := range []string{
					`action="` + tt.prefix + `/edit"`, `method="post"`,
					`hx-post="` + tt.prefix + `/edit"`, `hx-target="#workspace"`, `hx-swap="outerHTML"`,
					`<input type="hidden" name="file" value="booleans.cue">`,
					`<input type="hidden" name="entry" value="0">`,
					`<input type="hidden" name="field" value="Enabled">`,
					`<input type="hidden" name="view" value="entry">`,
				} {
					if !strings.Contains(form, want) {
						t.Errorf("inline form missing %q: %s", want, form)
					}
				}
				assertInlineBooleanToggle(t, form, tt.selected, tt.optional)
				for _, unwanted := range []string{`aria-label="Edit Enabled value"`, `aria-label="Edit Enabled"`, `type="radio"`, `<select`, `value=""`} {
					if strings.Contains(article, unwanted) {
						t.Errorf("inline boolean unexpectedly contains %q: %s", unwanted, article)
					}
				}
			}
			if !tt.readOnly && !tt.list && (!strings.Contains(article, `aria-label="Edit Name value"`) || !strings.Contains(article, `data-field-edit-trigger`)) {
				t.Errorf("nonboolean field lost its modal edit control: %s", article)
			}
		})
	}
}

func TestListBooleanLabels(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		declaration string
		value       string
		label       string
		readOnly    bool
		want        string
	}{
		{name: "writable true", declaration: "bool", value: "true", want: `<span class="has-text-info">✔</span>&nbsp;Enabled`},
		{name: "writable false", declaration: "bool", value: "false", want: `<span class="has-text-danger">✖</span>&nbsp;Enabled`},
		{name: "read only true", declaration: "bool", value: "true", readOnly: true, want: `<span class="has-text-info">✔</span>&nbsp;Enabled`},
		{name: "read only false", declaration: "bool", value: "false", readOnly: true, want: `<span class="has-text-danger">✖</span>&nbsp;Enabled`},
		{name: "unset optional", declaration: "bool", want: `<span class="has-text-grey">☐</span>&nbsp;Enabled`},
		{name: "unset read only", declaration: "bool", readOnly: true, want: `<span class="has-text-grey">☐</span>&nbsp;Enabled`},
		{name: "unset optional ignores true default", declaration: "*true | bool", want: `<span class="has-text-grey">☐</span>&nbsp;Enabled`},
		{name: "unset read only ignores true default", declaration: "*true | bool", readOnly: true, want: `<span class="has-text-grey">☐</span>&nbsp;Enabled`},

		{name: "label is escaped", declaration: "bool", value: "false", label: "<Enabled & ready>", want: `<span class="has-text-danger">✖</span>&nbsp;&lt;Enabled &amp; ready&gt;`},
		{name: "secret remains masked", declaration: "bool @cuebook(argon2id)", value: "true", want: "••••••••"},
		{name: "read only secret remains masked", declaration: "bool @cuebook(argon2id)", value: "false", readOnly: true, want: "••••••••"},
		{name: "string false is not a boolean", declaration: "string", value: `"false"`, want: "false"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			label := tt.label
			if label == "" {
				label = "Enabled"
			}
			quotedLabel := strconv.Quote(label)
			entry := `Name: "Existing"`
			if tt.value != "" {
				entry += ", " + quotedLabel + ": " + tt.value
			}
			source := "#entry: {Name: string @cuebook(title), " + quotedLabel + "?: " + tt.declaration + "}\n[...#entry] & [{" + entry + "}]"
			files := fstest.MapFS{"booleans.cue": &fstest.MapFile{Data: []byte(source)}}
			var handler http.Handler
			var err error
			if tt.readOnly {
				handler, err = New(files)
			} else {
				handler, err = NewWithCommitter(files, mapFileCommitter{files: files})
			}
			if err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest(http.MethodGet, "http://example.test/?file=booleans.cue", nil)
			request.Header.Set("HX-Request", "true")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				t.Fatalf("GET status = %d; body: %s", response.Code, response.Body.String())
			}
			article := inlineBooleanArticle(t, response.Body.String())
			if want := "<output>" + tt.want + "</output>"; !strings.Contains(article, want) {
				t.Errorf("list does not contain %q: %s", want, article)
			}
			if strings.Contains(article, "boolean-field-form") || strings.Contains(article, "boolean-toggle") {
				t.Errorf("list unexpectedly contains entry-view toggle buttons: %s", article)
			}
			if tt.readOnly {
				if strings.Contains(article, "data-field-edit-trigger") {
					t.Errorf("read-only list contains modal edit controls: %s", article)
				}
			} else if !strings.Contains(article, `aria-label="Edit `+label+` value"`) && tt.label == "" {
				t.Errorf("writable list lost its modal edit control: %s", article)
			}
		})
	}
}

func TestInlineBooleanSubmission(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		declaration string
		existing    string
		value       string
		prefix      string
		optional    bool
	}{
		{name: "false to true", declaration: "Enabled: bool", existing: "false", value: "true"},
		{name: "true to false mounted", declaration: "Enabled: bool", existing: "true", value: "false", prefix: "/catalog"},
		{name: "unset optional detail to true", declaration: "Enabled?: bool @cuebook(detail)", value: "true", optional: true},
		{name: "optional true to false", declaration: "Enabled?: bool", existing: "true", value: "false", optional: true},
		{name: "optional false to unset", declaration: "Enabled?: bool", existing: "false", value: "not_set", optional: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			entry := `Name: "Existing"`
			if tt.existing != "" {
				entry += ", Enabled: " + tt.existing
			}
			source := "#entry: {\nName: string @cuebook(title)\n" + tt.declaration + "\n}\n[...#entry] & [{Name: \"Unrelated entry\", Enabled: false}, {" + entry + "}]"
			files := fstest.MapFS{"booleans.cue": &fstest.MapFile{Data: []byte(source)}}
			handler, err := NewWithCommitter(files, mapFileCommitter{files: files}, WithServeMuxPrefix(tt.prefix))
			if err != nil {
				t.Fatal(err)
			}
			originalURL := inlineBooleanEntryURL(t, files, 1, tt.prefix)
			values := url.Values{
				"file": {"booleans.cue"}, "entry": {"1"}, "field": {"Enabled"},
				"view": {"entry"}, "value": {tt.value},
			}
			request := httptest.NewRequest(http.MethodPost, "http://example.test"+tt.prefix+"/edit", strings.NewReader(values.Encode()))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			request.Header.Set("Origin", "http://example.test")
			request.Header.Set("HX-Request", "true")
			request.Header.Set("HX-Current-URL", "http://example.test"+originalURL)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				t.Fatalf("POST status = %d; body: %s", response.Code, response.Body.String())
			}
			book, err := cuebook.New(files["booleans.cue"].Data)
			if err != nil {
				t.Fatalf("invalid saved document: %v", err)
			}
			updated, err := book.GetValue(1)
			if err != nil {
				t.Fatal(err)
			}
			enabled := updated.LookupPath(cue.ParsePath("Enabled"))
			if tt.value == "not_set" {
				if enabled.Exists() {
					t.Fatalf("unset optional Enabled was not removed: %s", files["booleans.cue"].Data)
				}
			} else if got, err := enabled.Bool(); err != nil || got != (tt.value == "true") {
				t.Fatalf("saved Enabled = %t, err = %v; want typed boolean %s", got, err, tt.value)
			}
			wantURL := inlineBooleanEntryURL(t, files, 1, tt.prefix)
			if wantURL == originalURL {
				t.Fatal("test edit did not change the entry byte range")
			}
			if got := response.Header().Get("HX-Replace-Url"); got != wantURL {
				t.Errorf("HX-Replace-Url = %q, want %q", got, wantURL)
			}
			for _, header := range []string{"HX-Redirect", "HX-Location", "Location"} {
				if got := response.Header().Get(header); got != "" {
					t.Errorf("unexpected %s = %q", header, got)
				}
			}
			body := response.Body.String()
			for _, want := range []string{`<main id="workspace"`, `id="entry-1"`, `class="boolean-field-form"`, `name="view" value="entry"`} {
				if !strings.Contains(body, want) {
					t.Errorf("entry response missing %q", want)
				}
			}
			if strings.Contains(body, "Unrelated entry") || strings.Contains(body, "<!doctype html>") {
				t.Error("save did not preserve the single-entry workspace fragment")
			}
			article := inlineBooleanArticle(t, body)
			assertInlineBooleanToggle(t, article, tt.value, tt.optional)
			get := httptest.NewRequest(http.MethodGet, "http://example.test"+wantURL, nil)
			get.Header.Set("HX-Request", "true")
			refreshed := httptest.NewRecorder()
			handler.ServeHTTP(refreshed, get)
			if refreshed.Code != http.StatusOK {
				t.Fatalf("replacement URL status = %d; body: %s", refreshed.Code, refreshed.Body.String())
			}
			assertInlineBooleanToggle(t, inlineBooleanArticle(t, refreshed.Body.String()), tt.value, tt.optional)
		})
	}
}

func assertInlineBooleanToggle(t *testing.T, markup, state string, optional bool) {
	t.Helper()
	forms := regexp.MustCompile(`(?s)<form class="boolean-field-form"[^>]*>.*?</form>`).FindAllString(markup, -1)
	if len(forms) != 1 {
		t.Fatalf("inline boolean form count = %d, want 1: %s", len(forms), markup)
	}
	markup = forms[0]
	class, pressed, next, symbol := "button boolean-toggle", state, "true", "✖"
	switch state {
	case "true":
		class, next, symbol = class+" has-text-info", "false", "✔"
	case "false":
		if optional {
			next = "not_set"
		}
	case "not_set":
		pressed, symbol = "mixed", "☐"
	default:
		t.Fatalf("unknown boolean state %q", state)
	}
	if strings.Count(markup, "<button ") != 1 {
		t.Errorf("want a single boolean toggle button: %s", markup)
	}
	want := `<button class="` + class + `" type="submit" name="value" value="` + next + `" aria-pressed="` + pressed + `" aria-label="Enabled" title="Enabled: ` + state + `; set to ` + next + `">` + symbol + `</button>`
	if !strings.Contains(markup, want) {
		t.Errorf("inline boolean missing toggle %q: %s", want, markup)
	}
	for _, unwanted := range []string{`type="radio"`, `<select`, `type="hidden" name="value"`} {
		if strings.Contains(markup, unwanted) {
			t.Errorf("inline boolean unexpectedly contains %q: %s", unwanted, markup)
		}
	}
}

func inlineBooleanEntryURL(t *testing.T, files fstest.MapFS, index int, prefix string) string {
	t.Helper()
	book, err := cuebook.New(files["booleans.cue"].Data)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := book.GetValue(index)
	if err != nil {
		t.Fatal(err)
	}
	span, err := cuebook.NewByteRange(entry)
	if err != nil {
		t.Fatal(err)
	}
	return prefix + entryURL("booleans.cue", span)
}

func inlineBooleanArticle(t *testing.T, body string) string {
	t.Helper()
	start := strings.Index(body, "<article ")
	end := strings.Index(body, "</article>")
	if start < 0 || end < start {
		t.Fatalf("entry article not found: %s", body)
	}
	return body[start:end]
}
