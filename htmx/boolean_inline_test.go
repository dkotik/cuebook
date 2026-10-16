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
		readOnly    bool
		list        bool
		secret      bool
	}{
		{name: "stored false", declaration: "Enabled: bool", existing: "false", selected: "false"},
		{name: "stored true", declaration: "Enabled: bool", existing: "true", selected: "true"},
		{name: "abstract optional bool defaults false", declaration: "Enabled?: bool", selected: "false"},
		{name: "unset optional detail defaults false", declaration: "Enabled?: bool @cuebook(detail)", selected: "false"},
		{name: "mounted routes", declaration: "Enabled: bool", existing: "true", prefix: "/catalog", selected: "true"},
		{name: "read only stays output", declaration: "Enabled: bool", existing: "true", readOnly: true},
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
			if tt.selected == "" {
				if len(forms) != 0 || strings.Contains(article, "boolean-toggle") {
					t.Errorf("unexpected inline boolean controls: %s", article)
				}
				if tt.readOnly {
					if !strings.Contains(article, "<output>true</output>") || strings.Contains(article, "data-field-edit-trigger") {
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
					`<div class="buttons has-addons boolean-toggle" role="group" aria-label="Enabled">`,
				} {
					if !strings.Contains(form, want) {
						t.Errorf("inline form missing %q: %s", want, form)
					}
				}
				if strings.Count(form, "<button ") != 2 || strings.Count(form, "is-link is-selected") != 1 {
					t.Errorf("want two buttons with exactly one selected: %s", form)
				}
				for _, choice := range []string{"false", "true"} {
					class := "button"
					if choice == tt.selected {
						class += " is-link is-selected"
					}
					want := `<button class="` + class + `" type="submit" name="value" value="` + choice + `" aria-pressed="` + strconv.FormatBool(choice == tt.selected) + `" aria-label="Set Enabled to ` + choice + `">`
					if !strings.Contains(form, want) {
						t.Errorf("inline form missing button %q: %s", want, form)
					}
				}
				for _, unwanted := range []string{`aria-label="Edit Enabled value"`, `aria-label="Edit Enabled"`, `type="radio"`, `value=""`} {
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
		{name: "writable true", declaration: "bool", value: "true", want: "✔\u00a0Enabled"},
		{name: "writable false", declaration: "bool", value: "false", want: "✖\u00a0Enabled"},
		{name: "read only true", declaration: "bool", value: "true", readOnly: true, want: "✔\u00a0Enabled"},
		{name: "read only false", declaration: "bool", value: "false", readOnly: true, want: "✖\u00a0Enabled"},
		{name: "unset optional defaults false", declaration: "bool", want: "✖\u00a0Enabled"},
		{name: "unset read only defaults false", declaration: "bool", readOnly: true, want: "✖\u00a0Enabled"},

		{name: "label is escaped", declaration: "bool", value: "false", label: "<Enabled & ready>", want: "✖\u00a0&lt;Enabled &amp; ready&gt;"},
		{name: "secret remains masked", declaration: "bool @cuebook(argon2id)", value: "true", want: "••••••••"},
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
			if strings.Contains(article, "boolean-field-form") {
				t.Errorf("list unexpectedly contains entry-view toggle buttons: %s", article)
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
	}{
		{name: "false to true", declaration: "Enabled: bool", existing: "false", value: "true"},
		{name: "true to false mounted", declaration: "Enabled: bool", existing: "true", value: "false", prefix: "/catalog"},
		{name: "unset optional detail to true", declaration: "Enabled?: bool @cuebook(detail)", value: "true"},
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
			if got, err := updated.LookupPath(cue.ParsePath("Enabled")).Bool(); err != nil || got != (tt.value == "true") {
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
			want := `value="` + tt.value + `" aria-pressed="true" aria-label="Set Enabled to ` + tt.value + `">`
			if !strings.Contains(article, want) {
				t.Errorf("saved value is not selected: %s", article)
			}
			get := httptest.NewRequest(http.MethodGet, "http://example.test"+wantURL, nil)
			get.Header.Set("HX-Request", "true")
			refreshed := httptest.NewRecorder()
			handler.ServeHTTP(refreshed, get)
			if refreshed.Code != http.StatusOK || !strings.Contains(inlineBooleanArticle(t, refreshed.Body.String()), want) {
				t.Errorf("replacement URL does not render the saved entry: status %d; %s", refreshed.Code, refreshed.Body.String())
			}
		})
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
