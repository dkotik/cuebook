package htmx

import (
	"bytes"
	"html"
	"html/template"
	"regexp"
	"strings"
	"testing"
)

func TestFilePickerRendering(t *testing.T) {
	t.Parallel()
	const nestedFile = "nested/deep/target & friends.cue"
	fileNames := []string{"root.cue", nestedFile, "archive/old.cue"}
	tests := []struct {
		name     string
		current  string
		selected string

		disableCurrent bool
		empty          bool
	}{
		{name: "preselected root file", current: "root.cue", selected: "root.cue"},
		{name: "preselected nested file with escaped path", current: nestedFile, selected: nestedFile},
		{name: "move requires an explicit destination", current: "root.cue", disableCurrent: true},
		{name: "move may preselect a different file", current: "root.cue", selected: nestedFile, disableCurrent: true},
		{name: "disabled current file cannot be preselected", current: nestedFile, selected: nestedFile, disableCurrent: true},
		{name: "unknown selection is not checked", current: "root.cue", selected: "missing.cue"},
		{name: "empty file tree", empty: true},
	}
	templates, err := template.New("file-selector.html").Funcs(template.FuncMap{
		"route": func(target string) string { return target },
	}).ParseFS(templateFiles, "templates/*.html")
	if err != nil {
		t.Fatal(err)
	}
	optionPattern := regexp.MustCompile(`<input type="radio" name="([^"]+)" value="([^"]+)"([^>]*)>`)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			files := fileNames
			if tt.empty {
				files = nil
			}
			options := filePickerOptions{
				Legend:         "Destination file",
				InputName:      "destination",
				Selected:       tt.selected,
				DisableCurrent: tt.disableCurrent,
			}

			picker := makeFilePicker(files, tt.current, options)

			var rendered bytes.Buffer
			if err := templates.ExecuteTemplate(&rendered, "file-selector.html", picker); err != nil {
				t.Fatal(err)
			}
			body := rendered.String()
			if strings.Contains(body, "<a ") || strings.Contains(body, "hx-get=") {
				t.Error("picker files must select a value, not navigate")
			}
			if !tt.empty {
				for _, want := range []string{`data-node-path="nested"`, `data-node-path="nested/deep"`, `class="tree-children"`, `class="tree-node-toggle" type="button"`} {
					if !strings.Contains(body, want) {
						t.Errorf("nested picker tree missing %q", want)
					}
				}
			}
			inputs := optionPattern.FindAllStringSubmatch(body, -1)
			if len(inputs) != len(files) {
				t.Fatalf("radio options = %d, want %d", len(inputs), len(files))
			}
			for _, input := range inputs {
				file := html.UnescapeString(input[2])
				attrs := input[3]
				disabled := tt.disableCurrent && file == tt.current
				checked := !disabled && file == tt.selected
				if input[1] != options.InputName {
					t.Errorf("%q radio name = %q, want %q", file, input[1], options.InputName)
				}
				for _, attribute := range []struct {
					name string
					want bool
				}{{"disabled", disabled}, {"checked", checked}, {"required", !disabled}} {
					if got := strings.Contains(attrs, " "+attribute.name); got != attribute.want {
						t.Errorf("%q %s = %t, want %t", file, attribute.name, got, attribute.want)
					}
				}
				label := file
				if file == tt.current {
					label += " (current file)"
				}
				if !strings.Contains(attrs, `aria-label="`+html.EscapeString(label)+`"`) {
					t.Errorf("%q radio does not have its full accessible path", file)
				}
			}

			// Picker decoration must not turn the sidebar's files into controls.
			rendered.Reset()
			for _, node := range makeFileTree(files, tt.current) {
				if err := templates.ExecuteTemplate(&rendered, "file-tree-node.html", node); err != nil {
					t.Fatal(err)
				}
			}
			if strings.Contains(rendered.String(), `type="radio"`) || strings.Count(rendered.String(), `<a class="tree-file-link"`) != len(files) {
				t.Error("sidebar files must remain navigation links")
			}
		})
	}
}
