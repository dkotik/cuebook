package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServeFileLinker(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		filePath string
		want     string
	}{
		{
			name:     "root file list link",
			filePath: "nested/file name.cue",
			want:     "/?file=nested%2Ffile+name.cue",
		},
		{
			name:     "prefixed file list link",
			prefix:   "/catalog",
			filePath: "nested/file name.cue",
			want:     "/catalog/?file=nested%2Ffile+name.cue",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := (serveFileLinker{prefix: test.prefix}).GetFileLink(test.filePath)
			if err != nil {
				t.Fatalf("GetFileLink() error = %v", err)
			}
			if got != test.want {
				t.Errorf("GetFileLink() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestIsGitDirectory(t *testing.T) {
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

			if got := isGitDirectory(directory); got != test.wantGitDir {
				t.Errorf("isGitDirectory() = %t, want %t", got, test.wantGitDir)
			}
		})
	}
}

func TestServeDirectoryRejectsInvalidPort(t *testing.T) {
	tests := []struct {
		name string
		port int64
	}{
		{name: "below valid range", port: 0},
		{name: "above valid range", port: 65536},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := serveDirectory(t.TempDir(), test.port)
			if err == nil || !strings.Contains(err.Error(), "expected a value between 1 and 65535") {
				t.Errorf("serveDirectory() error = %v, want invalid-port error", err)
			}
		})
	}
}

func TestServeCommandRequiresExactlyOneDirectory(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "missing directory", args: []string{"cuebook", "serve"}},
		{name: "extra directory", args: []string{"cuebook", "serve", "first", "second"}},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := newCLICommand().Run(context.Background(), test.args)
			if err == nil || !strings.Contains(err.Error(), "serve requires exactly one directory argument") {
				t.Errorf("Run() error = %v, want directory argument error", err)
			}
		})
	}
}

func TestServeCommandRejectsInvalidPort(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "port below valid range",
			args: []string{"cuebook", "serve", "directory", "--port=0"},
			want: "invalid port 0",
		},
		{
			name: "port above valid range",
			args: []string{"cuebook", "serve", "directory", "--port=65536"},
			want: "invalid port 65536",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := newCLICommand().Run(context.Background(), test.args)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Errorf("Run() error = %v, want error containing %q", err, test.want)
			}
		})
	}
}
