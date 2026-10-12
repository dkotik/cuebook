//go:build integration

package git

import (
	"fmt"
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

type recordingFileLinker struct {
	paths []string
}

func (linker *recordingFileLinker) GetFileLink(path string) (string, error) {
	linker.paths = append(linker.paths, path)
	return "/file?path=" + url.QueryEscape(path), nil
}

func TestNewListsGitChanges(t *testing.T) {
	t.Parallel()

	repository := t.TempDir()
	initializeRepository(t, repository, map[string]string{
		"edited.txt":    "initial edit\n",
		"deleted.txt":   "remove me\n",
		"old-name.txt":  "rename me\n",
		"unchanged.txt": "keep me\n",
	})

	writeRepositoryFile(t, repository, "edited.txt", "staged edit\n")
	runGit(t, repository, "add", "--", "edited.txt")
	writeRepositoryFile(t, repository, "edited.txt", "unstaged edit\n")
	if err := os.Remove(filepath.Join(repository, "deleted.txt")); err != nil {
		t.Fatal(err)
	}
	writeRepositoryFile(t, repository, "added.txt", "staged addition\n")
	runGit(t, repository, "add", "--", "added.txt")
	writeRepositoryFile(t, repository, "new file <script>.txt", "remove me\nuntracked\n")
	runGit(t, repository, "mv", "--", "old-name.txt", "renamed.txt")

	linker := &recordingFileLinker{}
	handler, err := New(os.DirFS(repository), WithFileLinker(linker))
	if err != nil {
		t.Fatal(err)
	}

	changes := handler.Changes()
	wantLostLines := []string{"initial edit"}
	if got := handler.LostLines(); !reflect.DeepEqual(got, wantLostLines) {
		t.Errorf("lost lines = %#v, want %#v", got, wantLostLines)
	}
	want := map[string]string{
		"added.txt":             "Added",
		"deleted.txt":           "Deleted",
		"edited.txt":            "Modified",
		"new file <script>.txt": "Untracked",
		"renamed.txt":           "Renamed",
	}
	if len(changes) != len(want) {
		t.Fatalf("got %d changes: %#v, want %d", len(changes), changes, len(want))
	}
	gotPaths := make([]string, 0, len(changes))
	for _, change := range changes {
		gotPaths = append(gotPaths, change.Path)
		if gotStatus, ok := want[change.Path]; !ok || gotStatus != change.Status {
			t.Errorf("change %q has status %q, want %q", change.Path, change.Status, gotStatus)
		}
		if change.Link != "/file?path="+url.QueryEscape(change.Path) {
			t.Errorf("link for %q = %q", change.Path, change.Link)
		}
	}
	wantPaths := []string{"added.txt", "deleted.txt", "edited.txt", "new file <script>.txt", "renamed.txt"}
	if !reflect.DeepEqual(gotPaths, wantPaths) {
		t.Errorf("changed paths = %#v, want %#v", gotPaths, wantPaths)
	}
	if !reflect.DeepEqual(linker.paths, wantPaths) {
		t.Errorf("FileLinker paths = %#v, want %#v", linker.paths, wantPaths)
	}

	changes[0].Path = "mutated"
	if handler.Changes()[0].Path != wantPaths[0] {
		t.Fatal("Changes() exposed the handler's internal snapshot")
	}

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/changes", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("ServeHTTP status = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Body.String(), `&lt;script&gt;.txt`) {
		t.Errorf("HTML did not escape the changed filename: %s", response.Body.String())
	}
}

func TestNewListsCleanTree(t *testing.T) {
	t.Parallel()

	repository := t.TempDir()
	initializeRepository(t, repository, map[string]string{"tracked.txt": "unchanged\n"})
	handler, err := New(os.DirFS(repository), WithFileLinker(&recordingFileLinker{}))
	if err != nil {
		t.Fatal(err)
	}
	if changes := handler.Changes(); len(changes) != 0 {
		t.Fatalf("Changes() = %#v, want an empty list", changes)
	}
	if lostLines := handler.LostLines(); len(lostLines) != 0 {
		t.Fatalf("LostLines() = %#v, want no lost lines", lostLines)
	}

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/changes", nil))
	if !strings.Contains(response.Body.String(), "No changes.") {
		t.Fatalf("clean-tree response = %q, want empty-state message", response.Body.String())
	}
}

func initializeRepository(t *testing.T, directory string, files map[string]string) {
	t.Helper()
	runGit(t, "", "init", "-q", directory)
	runGit(t, directory, "config", "user.name", "Git package test")
	runGit(t, directory, "config", "user.email", "git-package@example.invalid")
	for name, content := range files {
		writeRepositoryFile(t, directory, name, content)
	}
	runGit(t, directory, "add", "--all")
	runGit(t, directory, "commit", "-q", "-m", "initial")
}

func writeRepositoryFile(t *testing.T, directory, name, content string) {
	t.Helper()
	filePath := filepath.Join(directory, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func runGit(t *testing.T, directory string, arguments ...string) {
	t.Helper()
	if directory != "" {
		arguments = append([]string{"-C", directory}, arguments...)
	}
	command := exec.Command("git", arguments...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(arguments, " "), err, output)
	}
}

func TestNewRejectsMissingGitMetadata(t *testing.T) {
	t.Parallel()

	_, err := New(os.DirFS(t.TempDir()), WithFileLinker(&recordingFileLinker{}))
	if err == nil || !strings.Contains(err.Error(), "inspect .git directory") {
		t.Fatalf("New() error = %v, want missing-.git error", err)
	}
}

func TestNewPropagatesFileLinkerError(t *testing.T) {
	t.Parallel()

	repository := t.TempDir()
	initializeRepository(t, repository, map[string]string{"changed.txt": "initial\n"})
	writeRepositoryFile(t, repository, "changed.txt", "modified\n")
	linkerErr := fmt.Errorf("link unavailable")
	linker := failingFileLinker{err: linkerErr}
	_, err := New(os.DirFS(repository), WithFileLinker(linker))
	if err == nil || !strings.Contains(err.Error(), linkerErr.Error()) {
		t.Fatalf("New() error = %v, want propagated linker error", err)
	}
}

type failingFileLinker struct {
	err error
}

func (linker failingFileLinker) GetFileLink(string) (string, error) {
	return "", linker.err
}
