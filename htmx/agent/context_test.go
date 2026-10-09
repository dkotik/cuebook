package agent

import (
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
	"unicode/utf8"
)

func TestLoadContextSupportsFormatsAndRanksMatches(t *testing.T) {
	t.Parallel()

	source := fstest.MapFS{
		"data/person.CUE":        {Data: []byte("name: Ada\n")},
		"config/app.TOML":        {Data: []byte("name = 'Ada'\n")},
		"data/person.json":       {Data: []byte(`{"name":"Ada"}`)},
		"data/settings.yaml":     {Data: []byte("name: Ada\n")},
		"data/settings.yml":      {Data: []byte("name: Ada\n")},
		"notes/project.MD":       {Data: []byte("Ada's project notes")},
		"notes/archive.markdown": {Data: []byte("old notes")},
		"README.txt":             {Data: []byte("Plain text")},
		"ignored.csv":            {Data: []byte("not context")},
		"ignored/link.txt":       {Data: []byte("not followed"), Mode: fs.ModeSymlink},
	}
	configured := defaultOptions()
	index, err := loadContext(source, configured)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(index.files), 8; got != want {
		t.Fatalf("loaded %d files, want %d: %#v", got, want, index.files)
	}
	formats := make(map[string]string, len(index.files))
	for _, file := range index.files {
		formats[file.path] = file.format
	}
	for path, want := range map[string]string{
		"data/person.CUE":        "CUE",
		"config/app.TOML":        "TOML",
		"data/person.json":       "JSON",
		"data/settings.yaml":     "YAML",
		"data/settings.yml":      "YAML",
		"notes/project.MD":       "Markdown",
		"notes/archive.markdown": "Markdown",
		"README.txt":             "text",
	} {
		if got := formats[path]; got != want {
			t.Errorf("format for %q = %q, want %q", path, got, want)
		}
	}

	configured.maxContextFiles = 1
	selected, sources := index.selectFor("project notes", configured)
	if len(selected) != 1 || len(sources) != 1 || sources[0] != "notes/project.MD" {
		t.Fatalf("selected context = %#v, sources = %#v; want project.MD", selected, sources)
	}
	if !strings.Contains(selected[0].content, "(Markdown)") || !strings.Contains(selected[0].content, "Ada's project notes") {
		t.Errorf("selected file context lacks format/content: %q", selected[0].content)
	}
}

func TestLoadContextHonorsFileAndIndexByteLimits(t *testing.T) {
	t.Parallel()

	source := fstest.MapFS{
		"a.txt":     {Data: []byte("1234")},
		"b.txt":     {Data: []byte("5678")},
		"large.txt": {Data: []byte("12345")},
	}
	configured := defaultOptions()
	configured.maxFileBytes = 4
	configured.maxIndexedBytes = 6
	index, err := loadContext(source, configured)
	if err != nil {
		t.Fatal(err)
	}
	if len(index.files) != 1 || index.files[0].path != "a.txt" {
		t.Fatalf("loaded context files = %#v, want only a.txt", index.files)
	}
}

func TestContextBudgetTruncatesAtUTF8Boundary(t *testing.T) {
	t.Parallel()

	index := contextIndex{files: []contextFile{{path: "unicode.txt", format: "text", content: "café café café"}}}
	configured := defaultOptions()
	configured.maxContextFiles = 1
	configured.maxContextBytes = len("\n--- source: unicode.txt (text) ---\n") + len("caf") + 1
	selected, sources := index.selectFor("unicode", configured)
	if len(selected) != 1 || len(sources) != 1 {
		t.Fatalf("selection = %#v, sources = %#v", selected, sources)
	}
	if !strings.HasPrefix(selected[0].content, "\n--- source: unicode.txt (text) ---\n") {
		t.Fatalf("missing source header: %q", selected[0].content)
	}
	if !utf8.ValidString(selected[0].content) {
		t.Fatalf("context is not valid UTF-8: %q", selected[0].content)
	}
	if len(selected[0].content) > configured.maxContextBytes {
		t.Errorf("selected context is %d bytes, limit %d", len(selected[0].content), configured.maxContextBytes)
	}
}

func TestFileFormatIgnoresUnsupportedExtensions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		path   string
		format string
		want   bool
	}{
		{name: "CUE", path: "a.cue", format: "CUE", want: true},
		{name: "TOML uppercase", path: "a.TOML", format: "TOML", want: true},
		{name: "YAML alias", path: "a.yml", format: "YAML", want: true},
		{name: "Markdown alias", path: "a.markdown", format: "Markdown", want: true},
		{name: "unsupported CSV", path: "a.csv"},
		{name: "extensionless", path: "README"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			gotFormat, got := fileFormat(test.path)
			if got != test.want || gotFormat != test.format {
				t.Errorf("fileFormat(%q) = (%q, %v), want (%q, %v)", test.path, gotFormat, got, test.format, test.want)
			}
		})
	}
}
