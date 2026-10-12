package git

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// FileChange describes one path reported as changed by git status.
// FileChange is one path reported as changed by Git, with a display status and link.
type FileChange struct {
	Path   string
	Status string
	Link   string
}

// Handler serves an HTML fragment containing a snapshot of changed files.
// Handler serves a snapshot of changed files as an HTML fragment.
type Handler struct {
	changes []FileChange
}

var changeListTemplate = template.Must(template.New("git-changes").Parse(`<ul class="git-changes" aria-label="Git changes">
{{range .}}
  <li class="git-change"><span class="git-change-status">{{.Status}}</span> <a href="{{.Link}}">{{.Path}}</a></li>
{{else}}
  <li class="git-changes-empty">No changes.</li>
{{end}}
</ul>
`))

// New snapshots source into a temporary working tree, reads its Git status, and
// creates a handler for the changed-file list. source must contain a standard
// .git directory. The resulting list does not change after New returns.
func New(source fs.FS, opts ...Option) (*Handler, error) {
	if isNil(source) {
		return nil, errors.New("git: source filesystem is nil")
	}
	configured, err := resolveOptions(opts)
	if err != nil {
		return nil, err
	}
	paths, err := changedPaths(source)
	if err != nil {
		return nil, err
	}

	changes := make([]FileChange, 0, len(paths))
	for _, path := range paths {
		link, err := configured.FileLinker.GetFileLink(path.path)
		if err != nil {
			return nil, fmt.Errorf("git: create link for %q: %w", path.path, err)
		}
		link = strings.TrimSpace(link)
		if link == "" {
			return nil, fmt.Errorf("git: file linker returned an empty link for %q", path.path)
		}
		changes = append(changes, FileChange{Path: path.path, Status: path.status, Link: link})
	}
	return &Handler{changes: changes}, nil
}

// Changes returns a copy of the changed-file snapshot.
func (handler *Handler) Changes() []FileChange {
	return append([]FileChange(nil), handler.changes...)
}

// ServeHTTP writes the changed-file list as an HTML fragment.
func (handler *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "Method not allowed.", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if err := changeListTemplate.Execute(w, handler.changes); err != nil {
		http.Error(w, "Unable to render Git changes.", http.StatusInternalServerError)
	}
}

type changedPath struct {
	path   string
	status string
}

func changedPaths(source fs.FS) ([]changedPath, error) {
	gitDirectory, err := fs.Stat(source, ".git")
	if err != nil {
		return nil, fmt.Errorf("git: inspect .git directory: %w", err)
	}
	if !gitDirectory.IsDir() {
		return nil, errors.New("git: source must contain a .git directory; linked worktrees are not supported")
	}

	workingTree, err := os.MkdirTemp("", "cuebook-git-")
	if err != nil {
		return nil, fmt.Errorf("git: create temporary working tree: %w", err)
	}
	defer os.RemoveAll(workingTree)

	if err := copyFilesystem(source, workingTree); err != nil {
		return nil, err
	}

	workingTree, err = filepath.Abs(workingTree)
	if err != nil {
		return nil, fmt.Errorf("git: resolve temporary working tree: %w", err)
	}
	command := exec.Command(
		"git",
		"--no-optional-locks",
		"-C", workingTree,
		"-c", "core.fsmonitor=false",
		"-c", "core.untrackedCache=false",
		"status", "--porcelain=v1", "-z", "--untracked-files=all",
	)
	command.Env = gitEnvironment(workingTree)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	output, err := command.Output()
	if err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail != "" {
			return nil, fmt.Errorf("git: read working-tree status: %w: %s", err, detail)
		}
		return nil, fmt.Errorf("git: read working-tree status: %w", err)
	}
	return parseStatus(output)
}

func copyFilesystem(source fs.FS, destination string) error {
	err := fs.WalkDir(source, ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("git: walk source at %q: %w", name, walkErr)
		}
		if name == "." {
			return nil
		}
		if !fs.ValidPath(name) || !filepath.IsLocal(filepath.FromSlash(name)) {
			return fmt.Errorf("git: invalid source path %q", name)
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return fmt.Errorf("git: source contains unsupported symlink %q", name)
		}

		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("git: inspect source path %q: %w", name, err)
		}
		target := filepath.Join(destination, filepath.FromSlash(name))
		if info.IsDir() {
			mode := info.Mode().Perm()
			if mode == 0 {
				mode = 0o755
			}
			if err := os.MkdirAll(target, mode); err != nil {
				return fmt.Errorf("git: create temporary directory for %q: %w", name, err)
			}
			return nil
		}
		if !info.Mode().IsRegular() {
			return nil
		}

		mode := info.Mode().Perm()
		if mode == 0 {
			mode = 0o644
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("git: create parent directory for %q: %w", name, err)
		}
		input, err := source.Open(name)
		if err != nil {
			return fmt.Errorf("git: open source file %q: %w", name, err)
		}
		output, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
		if err != nil {
			_ = input.Close()
			return fmt.Errorf("git: create temporary file for %q: %w", name, err)
		}
		_, copyErr := io.Copy(output, input)
		inputCloseErr := input.Close()
		outputCloseErr := output.Close()
		if err := errors.Join(copyErr, inputCloseErr, outputCloseErr); err != nil {
			return fmt.Errorf("git: copy source file %q: %w", name, err)
		}
		if err := os.Chmod(target, mode); err != nil {
			return fmt.Errorf("git: set permissions for %q: %w", name, err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func gitEnvironment(workingTree string) []string {
	environment := make([]string, 0, len(os.Environ())+5)
	for _, variable := range os.Environ() {
		name, _, _ := strings.Cut(variable, "=")
		if strings.HasPrefix(strings.ToUpper(name), "GIT_") {
			continue
		}
		environment = append(environment, variable)
	}
	return append(environment,
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL="+os.DevNull,
		"GIT_DIR="+filepath.Join(workingTree, ".git"),
		"GIT_WORK_TREE="+workingTree,
		"GIT_OPTIONAL_LOCKS=0",
		"GIT_TERMINAL_PROMPT=0",
		"GIT_PAGER=cat",
	)
}

func parseStatus(output []byte) ([]changedPath, error) {
	byPath := make(map[string][]string)
	records := bytes.Split(output, []byte{0})
	for index := 0; index < len(records); index++ {
		record := records[index]
		if len(record) == 0 {
			continue
		}
		if len(record) < 4 || record[2] != ' ' {
			return nil, fmt.Errorf("git: malformed porcelain status record %q", record)
		}
		code := string(record[:2])
		path := string(record[3:])
		if !fs.ValidPath(path) || !filepath.IsLocal(filepath.FromSlash(path)) {
			return nil, fmt.Errorf("git: invalid changed path %q", path)
		}
		addStatus(byPath, path, statusLabel(code))

		if strings.ContainsAny(code, "RC") {
			index++
			if index >= len(records) || len(records[index]) == 0 {
				return nil, fmt.Errorf("git: missing source path for rename or copy of %q", path)
			}
			// In -z output, the following NUL-delimited field is the old path.
			// Validate it, but link only the destination path above.
			oldPath := string(records[index])
			if !fs.ValidPath(oldPath) || !filepath.IsLocal(filepath.FromSlash(oldPath)) {
				return nil, fmt.Errorf("git: invalid source path %q", oldPath)
			}
		}
	}

	paths := make([]string, 0, len(byPath))
	for path := range byPath {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	result := make([]changedPath, 0, len(paths))
	for _, path := range paths {
		result = append(result, changedPath{path: path, status: strings.Join(byPath[path], ", ")})
	}
	return result, nil
}

func addStatus(byPath map[string][]string, path, status string) {
	for _, existing := range byPath[path] {
		if existing == status {
			return
		}
	}
	byPath[path] = append(byPath[path], status)
}

func statusLabel(code string) string {
	if code == "??" {
		return "Untracked"
	}
	labels := make([]string, 0, 2)
	for _, status := range code {
		label := ""
		switch status {
		case 'A':
			label = "Added"
		case 'C':
			label = "Copied"
		case 'D':
			label = "Deleted"
		case 'M', 'T':
			label = "Modified"
		case 'R':
			label = "Renamed"
		case 'U':
			label = "Conflicted"
		}
		if label == "" {
			continue
		}
		found := false
		for _, existing := range labels {
			if existing == label {
				found = true
				break
			}
		}
		if !found {
			labels = append(labels, label)
		}
	}
	if len(labels) == 0 {
		return "Changed"
	}
	return strings.Join(labels, ", ")
}
