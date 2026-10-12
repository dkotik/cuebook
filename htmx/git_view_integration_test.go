//go:build integration

package htmx

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type gitViewFileLinker struct {
	paths []string
}

func (linker *gitViewFileLinker) GetFileLink(filePath string) (string, error) {
	linker.paths = append(linker.paths, filePath)
	return "/review?file=" + url.QueryEscape(filePath), nil
}

func TestGitChangesViewUsesSourceAndFileLinker(t *testing.T) {
	repository := t.TempDir()
	initializeHTMXGitRepository(t, repository, "nested/changed.txt", "before\n")

	linker := &gitViewFileLinker{}
	handler, err := New(os.DirFS(repository), WithGitChanges(linker), WithServeMuxPrefix("/catalog"))
	if err != nil {
		t.Fatal(err)
	}
	writeHTMXGitFile(t, repository, "nested/changed.txt", "after\n")

	tests := []struct {
		name      string
		hxRequest bool
		contains  []string
		omits     []string
	}{
		{
			name: "full page",
			contains: []string{
				"<!doctype html>",
				`<nav class="file-nav menu"`,
				`href="/catalog/changes" hx-get="/catalog/changes"`,
				`aria-current="page"`,
				`<main id="workspace"`,
				"Changes",
				"Modified",
				`<a href="/review?file=nested%2Fchanged.txt">nested/changed.txt</a>`,
			},
		},
		{
			name:      "HTMX workspace fragment",
			hxRequest: true,
			contains: []string{
				`<main id="workspace"`,
				"Changes",
				"Modified",
				`<a href="/review?file=nested%2Fchanged.txt">nested/changed.txt</a>`,
			},
			omits: []string{"<!doctype html>", `class="file-nav menu"`},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "http://example.test/catalog/changes", nil)
			if test.hxRequest {
				request.Header.Set("HX-Request", "true")
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
			}
			body := response.Body.String()
			for _, expected := range test.contains {
				if !strings.Contains(body, expected) {
					t.Errorf("body does not contain %q: %s", expected, body)
				}
			}
			for _, absent := range test.omits {
				if strings.Contains(body, absent) {
					t.Errorf("body unexpectedly contains %q: %s", absent, body)
				}
			}
		})
	}
	wantPaths := []string{"nested/changed.txt", "nested/changed.txt"}
	if !reflect.DeepEqual(linker.paths, wantPaths) {
		t.Fatalf("FileLinker paths = %#v, want %#v", linker.paths, wantPaths)
	}
}

func initializeHTMXGitRepository(t *testing.T, repository, filePath, content string) {
	t.Helper()
	runHTMXGit(t, repository, "init", "-q")
	runHTMXGit(t, repository, "config", "user.name", "HTMX Git view test")
	runHTMXGit(t, repository, "config", "user.email", "htmx-git-view@example.invalid")
	writeHTMXGitFile(t, repository, filePath, content)
	runHTMXGit(t, repository, "add", "--all")
	runHTMXGit(t, repository, "commit", "-q", "-m", "initial")
}

func writeHTMXGitFile(t *testing.T, repository, filePath, content string) {
	t.Helper()
	fullPath := filepath.Join(repository, filepath.FromSlash(filePath))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func runHTMXGit(t *testing.T, repository string, arguments ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", repository}, arguments...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(arguments, " "), err, output)
	}
}
