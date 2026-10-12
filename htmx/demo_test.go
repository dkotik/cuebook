package htmx

import (
	"os"
	"path/filepath"
	"testing"
)

type fixedDemoFileLinker string

func (linker fixedDemoFileLinker) GetFileLink(string) (string, error) {
	return string(linker), nil
}

func TestDemoGitChangesOption(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		custom   bool
		wantLink string
	}{
		{
			name:     "root file list link",
			wantLink: "/?file=nested%2Ffile+name.cue",
		},
		{
			name:     "prefixed file list link",
			prefix:   "/catalog",
			wantLink: "/catalog/?file=nested%2Ffile+name.cue",
		},
		{
			name:     "keeps a caller-provided linker",
			prefix:   "/catalog",
			custom:   true,
			wantLink: "/custom-link",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			configured := &options{ServeMuxPrefix: test.prefix}
			if test.custom {
				configured.GitFileLinker = fixedDemoFileLinker("/custom-link")
			}
			if err := demoGitChangesOption()(configured); err != nil {
				t.Fatalf("demoGitChangesOption() error = %v", err)
			}
			got, err := configured.GitFileLinker.GetFileLink("nested/file name.cue")
			if err != nil {
				t.Fatalf("GetFileLink() error = %v", err)
			}
			if got != test.wantLink {
				t.Errorf("GetFileLink() = %q, want %q", got, test.wantLink)
			}
		})
	}
}

func TestIsGitDemoDirectory(t *testing.T) {
	tests := []struct {
		name       string
		gitDir     bool
		gitFile    bool
		wantGitDir bool
	}{
		{name: "git directory", gitDir: true, wantGitDir: true},
		{name: "linked worktree metadata file", gitFile: true},
		{name: "not a repository"},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			directory := t.TempDir()
			gitPath := filepath.Join(directory, ".git")
			if test.gitDir {
				if err := os.Mkdir(gitPath, 0o700); err != nil {
					t.Fatal(err)
				}
			}
			if test.gitFile {
				if err := os.WriteFile(gitPath, []byte("gitdir: ../.git/worktrees/demo\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			}

			if got := isGitDemoDirectory(directory); got != test.wantGitDir {
				t.Errorf("isGitDemoDirectory() = %t, want %t", got, test.wantGitDir)
			}
		})
	}
}
