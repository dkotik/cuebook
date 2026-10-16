package htmx

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"cuelang.org/go/cue"
	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/patch"
)

type booleanUnsetCommitter struct {
	mapFileCommitter
	calls int
}

func (c *booleanUnsetCommitter) Commit(name string, change patch.Patch) error {
	c.calls++
	return c.mapFileCommitter.Commit(name, change)
}

func TestEditBooleanUnset(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		declaration string
		existing    string
		values      []string
		want        string
		wantCalls   int
		rejected    bool
	}{
		{name: "true to unset", declaration: "Enabled?: bool", existing: "true", values: []string{"not_set"}, wantCalls: 1},
		{name: "false to unset", declaration: "Enabled?: bool", existing: "false", values: []string{"not_set"}, wantCalls: 1},
		{name: "unset to true", declaration: "Enabled?: bool", values: []string{"true"}, want: "true", wantCalls: 1},
		{name: "unset to false", declaration: "Enabled?: bool", values: []string{"false"}, want: "false", wantCalls: 1},
		{name: "repeated unset", declaration: "Enabled?: bool", existing: "true", values: []string{"not_set", "not_set"}, wantCalls: 1},
		{name: "already unset", declaration: "Enabled?: bool", values: []string{"not_set"}},
		{name: "optional true default stays absent", declaration: "Enabled?: *true | bool", existing: "false", values: []string{"not_set"}, wantCalls: 1},
		{name: "absent optional default stays absent", declaration: "Enabled?: *true | bool", values: []string{"not_set"}},
		{name: "required true rejects", declaration: "Enabled: bool", existing: "true", values: []string{"not_set"}, want: "true", rejected: true},
		{name: "required false rejects", declaration: "Enabled: bool", existing: "false", values: []string{"not_set"}, want: "false", rejected: true},
		{name: "required default rejects", declaration: "Enabled: *true | bool", existing: "false", values: []string{"not_set"}, want: "false", rejected: true},
		{name: "required inherited default rejects", declaration: "Enabled: *true | bool", values: []string{"not_set"}, want: "true", rejected: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			item := `Name: "Existing", Other: false, Secret: "stored-secret"`
			if tt.existing != "" {
				item += ", Enabled: " + tt.existing
			}
			source := []byte("#entry: {Name: string @cuebook(title), Other: bool, Secret: string @cuebook(argon2id), Missing?: *true | bool, " + tt.declaration + "}\n[...#entry] & [{" + item + "}]")
			files := fstest.MapFS{"booleans.cue": &fstest.MapFile{Data: bytes.Clone(source)}}
			committer := &booleanUnsetCommitter{mapFileCommitter: mapFileCommitter{files: files}}
			handler, err := NewWithCommitter(files, committer)
			if err != nil {
				t.Fatal(err)
			}
			for _, input := range tt.values {
				values := editValues("booleans.cue", "0", "Enabled", input)
				values.Set("view", "entry")
				response := submitEdit(t, handler, true, values)
				if tt.rejected {
					if response.Code != http.StatusNoContent || !strings.Contains(response.Header().Get("HX-Trigger"), "required boolean field Enabled cannot be unset") {
						t.Fatalf("status = %d, trigger = %q; want required-field rejection", response.Code, response.Header().Get("HX-Trigger"))
					}
					if response.Header().Get("HX-Replace-Url") != "" {
						t.Fatal("rejection changed the URL")
					}
				} else {
					if response.Code != http.StatusOK {
						t.Fatalf("status = %d; body: %s; trigger: %s", response.Code, response.Body.String(), response.Header().Get("HX-Trigger"))
					}
					book, err := cuebook.New(files["booleans.cue"].Data)
					if err != nil {
						t.Fatal(err)
					}
					entry, err := book.GetValue(0)
					if err != nil {
						t.Fatal(err)
					}
					r, err := cuebook.NewByteRange(entry)
					if err != nil {
						t.Fatal(err)
					}
					if got, want := response.Header().Get("HX-Replace-Url"), entryURL("booleans.cue", r); got != want {
						t.Fatalf("replacement URL = %q, want %q", got, want)
					}
					if strings.Contains(response.Body.String(), `name="value" value="stored-secret"`) {
						t.Fatal("response exposed the secret in an editor")
					}
				}
			}
			if committer.calls != tt.wantCalls {
				t.Fatalf("commit calls = %d, want %d", committer.calls, tt.wantCalls)
			}
			if tt.wantCalls == 0 && !bytes.Equal(files["booleans.cue"].Data, source) {
				t.Fatal("no-op or rejection changed source")
			}
			book, err := cuebook.New(files["booleans.cue"].Data)
			if err != nil {
				t.Fatal(err)
			}
			entry, err := book.GetValue(0)
			if err != nil {
				t.Fatal(err)
			}
			enabled := entry.LookupPath(cue.ParsePath("Enabled"))
			if tt.want == "" {
				if enabled.Exists() {
					t.Fatalf("unset boolean still exists: %s", files["booleans.cue"].Data)
				}
			} else if got, err := enabled.Bool(); err != nil || got != (tt.want == "true") {
				t.Fatalf("Enabled = %v, err = %v; want %s", got, err, tt.want)
			}
			if entry.LookupPath(cue.ParsePath("Missing")).Exists() {
				t.Fatal("unrelated optional default was materialized")
			}
			if got, err := entry.LookupPath(cue.ParsePath("Other")).Bool(); err != nil || got {
				t.Fatalf("Other = %v, err = %v; want false", got, err)
			}
			if got, err := entry.LookupPath(cue.ParsePath("Secret")).String(); err != nil || got != "stored-secret" {
				t.Fatalf("Secret = %q, err = %v", got, err)
			}
		})
	}
}

func TestAddBooleanUnset(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		declaration    string
		initial        string
		listConstraint string
		rejected       bool
	}{
		{name: "optional omitted", declaration: "Enabled?: bool"},
		{name: "optional default omitted", declaration: "Enabled?: *true | bool"},
		{name: "populated optional schema remains optional", declaration: "Enabled?: bool", initial: `{Name: "Existing", Enabled: true}`},
		{name: "required rejected", declaration: "Enabled: bool", rejected: true},
		{name: "required default rejected", declaration: "Enabled: *true | bool", rejected: true},
		{name: "required list constraint rejects", declaration: "Enabled?: bool", listConstraint: " & [...{Enabled: bool}]", initial: `{Name: "Existing", Enabled: true}`, rejected: true},
		{name: "required default list constraint rejects", declaration: "Enabled?: bool", listConstraint: " & [...{Enabled: *true | bool}]", initial: `{Name: "Existing"}`, rejected: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			source := []byte("#entry: {Name: string @cuebook(title), " + tt.declaration + "}\n[...#entry]" + tt.listConstraint + " & [" + tt.initial + "]")
			files := fstest.MapFS{"booleans.cue": &fstest.MapFile{Data: bytes.Clone(source)}}
			committer := &booleanUnsetCommitter{mapFileCommitter: mapFileCommitter{files: files}}
			handler, err := NewWithCommitter(files, committer)
			if err != nil {
				t.Fatal(err)
			}
			response := submitAdd(t, handler, true, addEntryValues("booleans.cue", [][2]string{{"Name", "Added"}, {"Enabled", "not_set"}}))
			if tt.rejected {
				if response.Code != http.StatusInternalServerError || !strings.Contains(response.Body.String(), "required boolean field Enabled cannot be unset") {
					t.Fatalf("status = %d, body = %q", response.Code, response.Body.String())
				}
				if committer.calls != 0 || !bytes.Equal(files["booleans.cue"].Data, source) {
					t.Fatal("rejected add changed source")
				}
				return
			}
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d; body: %s; trigger: %s", response.Code, response.Body.String(), response.Header().Get("HX-Trigger"))
			}
			book, err := cuebook.New(files["booleans.cue"].Data)
			if err != nil {
				t.Fatal(err)
			}
			length, err := book.Len()
			if err != nil {
				t.Fatal(err)
			}
			wantLength := 1
			if tt.initial != "" {
				wantLength++
			}
			if length != wantLength || committer.calls != 1 {
				t.Fatalf("length = %d, commits = %d; want %d, 1", length, committer.calls, wantLength)
			}
			entry, err := book.GetValue(length - 1)
			if err != nil {
				t.Fatal(err)
			}
			if entry.LookupPath(cue.ParsePath("Enabled")).Exists() {
				t.Fatalf("unset optional boolean persisted: %s", files["booleans.cue"].Data)
			}
		})
	}
}

func TestFieldViewBooleanUnset(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, declaration, existing, want string
		constraint, listConstraint        string
		boolean, optional, secret         bool
	}{
		{name: "optional absent", declaration: "Enabled?: bool", want: "not_set", boolean: true, optional: true},
		{name: "optional abstract default", declaration: "Enabled?: *true | bool", want: "not_set", boolean: true, optional: true},
		{name: "optional false default stays unset", declaration: "Enabled?: *false | bool", want: "not_set", boolean: true, optional: true},
		{name: "optional default conjunction stays unset", declaration: "Enabled?: *true | bool", constraint: " & {Enabled?: bool}", want: "not_set", boolean: true, optional: true},
		{name: "optional default list conjunction stays unset", declaration: "Enabled?: *true | bool", listConstraint: " & [...{Enabled?: bool}]", want: "not_set", boolean: true, optional: true},
		{name: "optional true", declaration: "Enabled?: bool", existing: "true", want: "true", boolean: true, optional: true},
		{name: "optional false", declaration: "Enabled?: bool", existing: "false", want: "false", boolean: true, optional: true},
		{name: "required true", declaration: "Enabled: bool", existing: "true", want: "true", boolean: true},
		{name: "required false", declaration: "Enabled: bool", existing: "false", want: "false", boolean: true},
		{name: "required default", declaration: "Enabled: *true | bool", existing: "false", want: "false", boolean: true},
		{name: "inherited required default", declaration: "Enabled: *true | bool", want: "true", boolean: true},
		{name: "inherited required false default", declaration: "Enabled: *false | bool", want: "false", boolean: true},
		{name: "required list constraint overrides optional", declaration: "Enabled?: bool", listConstraint: " & [...{Enabled: bool}]", existing: "true", want: "true", boolean: true},
		{name: "required default list constraint overrides optional", declaration: "Enabled?: bool", listConstraint: " & [...{Enabled: *true | bool}]", want: "true", boolean: true},
		{name: "optional conjunction stays optional", declaration: "Enabled?: bool", constraint: " & {Enabled?: bool}", existing: "true", want: "true", boolean: true, optional: true},
		{name: "required conjunction overrides optional", declaration: "Enabled?: bool", constraint: " & {Enabled: bool}", existing: "true", want: "true", boolean: true},
		{name: "optional conjunction does not override required", declaration: "Enabled: bool", constraint: " & {Enabled?: bool}", existing: "false", want: "false", boolean: true},
		{name: "required default conjunction overrides optional", declaration: "Enabled?: bool", constraint: " & {Enabled: *true | bool}", want: "true", boolean: true},
		{name: "union is not boolean", declaration: "Enabled?: bool | string", optional: true},
		{name: "abstract secret retains empty value", declaration: "Enabled?: bool @cuebook(argon2id)", boolean: true, optional: true, secret: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			item := `Name: "Existing"`
			if tt.existing != "" {
				item += ", Enabled: " + tt.existing
			}
			book, err := cuebook.New([]byte("#entry: {Name: string, " + tt.declaration + "}" + tt.constraint + "\n[...#entry]" + tt.listConstraint + " & [{" + item + "}]"))
			if err != nil {
				t.Fatal(err)
			}
			value, err := book.GetValue(0)
			if err != nil {
				t.Fatal(err)
			}
			entry, err := cuebook.NewEntry(value)
			if err != nil {
				t.Fatal(err)
			}
			field, ok := entry.GetFieldByName("Enabled")
			if !ok {
				t.Fatal("Enabled not found")
			}
			view := makeFieldView(field, "booleans.cue", 0, false, false)
			if view.Value != tt.want || view.Boolean != tt.boolean || view.Secret != tt.secret {
				t.Fatalf("view value/boolean/secret = %q/%v/%v, want %q/%v/%v", view.Value, view.Boolean, view.Secret, tt.want, tt.boolean, tt.secret)
			}
			if wantEmpty := tt.want == "" || tt.want == "not_set"; view.Empty != wantEmpty {
				t.Fatalf("empty = %v, want %v", view.Empty, wantEmpty)
			}
			if got := entryFieldOptional(value, "Enabled"); got != tt.optional {
				t.Fatalf("optional = %v, want %v", got, tt.optional)
			}
		})
	}
}

func TestLiteralNotSetBooleanUnset(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, declaration string
		add               bool
	}{
		{name: "edit string", declaration: "Enabled?: string"},
		{name: "edit union", declaration: "Enabled?: bool | string"},
		{name: "edit secret", declaration: "Enabled?: string @cuebook(argon2id)"},
		{name: "add string", declaration: "Enabled?: string", add: true},
		{name: "add union", declaration: "Enabled?: bool | string", add: true},
		{name: "add secret", declaration: "Enabled?: string @cuebook(argon2id)", add: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			initial := `{Name: "Existing"}`
			if tt.add {
				initial = ""
			}
			files := fstest.MapFS{"booleans.cue": &fstest.MapFile{Data: []byte("#entry: {Name: string @cuebook(title), " + tt.declaration + "}\n[...#entry] & [" + initial + "]")}}
			handler, err := NewWithCommitter(files, mapFileCommitter{files: files})
			if err != nil {
				t.Fatal(err)
			}
			var response *httptest.ResponseRecorder
			if tt.add {
				response = submitAdd(t, handler, true, addEntryValues("booleans.cue", [][2]string{{"Name", "Added"}, {"Enabled", "not_set"}}))
			} else {
				response = submitEdit(t, handler, true, editValues("booleans.cue", "0", "Enabled", "not_set"))
			}
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d; trigger: %s", response.Code, response.Header().Get("HX-Trigger"))
			}
			book, err := cuebook.New(files["booleans.cue"].Data)
			if err != nil {
				t.Fatal(err)
			}
			entry, err := book.GetValue(0)
			if err != nil {
				t.Fatal(err)
			}
			got, err := entry.LookupPath(cue.ParsePath("Enabled")).String()
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(tt.declaration, "argon2id") {
				if got == "not_set" || !strings.HasPrefix(got, "$argon2id$") {
					t.Fatalf("secret was not hashed: %q", got)
				}
			} else if got != "not_set" {
				t.Fatalf("string = %q, want not_set", got)
			}
		})
	}
}
