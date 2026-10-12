package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func lostLinesForWorkingTree(workingTree string, changes []changedPath) ([]string, error) {
	var addedLines, removedLines []string
	diffs := []struct {
		name      string
		arguments []string
	}{
		{
			name:      "staged",
			arguments: []string{"diff", "--cached", "--no-ext-diff", "--no-color", "--no-renames", "--unified=0"},
		},
		{
			name:      "unstaged",
			arguments: []string{"diff", "--no-ext-diff", "--no-color", "--no-renames", "--unified=0"},
		},
	}
	for _, diff := range diffs {
		output, err := runGitOutput(workingTree, diff.arguments...)
		if err != nil {
			return nil, fmt.Errorf("git: read %s diff for lost lines: %w", diff.name, err)
		}
		added, removed := parseDiffLineChanges(output)
		addedLines = append(addedLines, added...)
		removedLines = append(removedLines, removed...)
	}

	for _, change := range changes {
		if change.status != "Untracked" {
			continue
		}
		filePath := filepath.Join(workingTree, filepath.FromSlash(change.path))
		contents, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("git: read untracked file %q for lost lines: %w", change.path, err)
		}
		addedLines = append(addedLines, linesFromFile(contents)...)
	}

	return countLostLines(addedLines, removedLines), nil
}

func runGitOutput(workingTree string, arguments ...string) ([]byte, error) {
	gitArguments := []string{
		"--no-optional-locks",
		"-C", workingTree,
		"-c", "core.fsmonitor=false",
		"-c", "core.untrackedCache=false",
	}
	command := exec.Command("git", append(gitArguments, arguments...)...)
	command.Env = gitEnvironment(workingTree)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	output, err := command.Output()
	if err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail != "" {
			return nil, fmt.Errorf("%w: %s", err, detail)
		}
		return nil, err
	}
	return output, nil
}

func parseDiffLineChanges(output []byte) (addedLines, removedLines []string) {
	inHunk := false
	for _, line := range strings.Split(string(output), "\n") {
		if strings.HasPrefix(line, "diff --git ") {
			inHunk = false
			continue
		}
		if strings.HasPrefix(line, "@@ ") {
			inHunk = true
			continue
		}
		if !inHunk || line == "" || strings.HasPrefix(line, "\\ ") {
			continue
		}
		switch line[0] {
		case '+':
			addedLines = append(addedLines, line[1:])
		case '-':
			removedLines = append(removedLines, line[1:])
		}
	}
	return addedLines, removedLines
}

func linesFromFile(contents []byte) []string {
	if len(contents) == 0 || bytes.IndexByte(contents, 0) >= 0 {
		return nil
	}
	lines := strings.Split(string(contents), "\n")
	if contents[len(contents)-1] == '\n' {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func countLostLines(addedLines []string, removedLines []string) (lostLines []string) {
	known := make(map[string]struct{}, len(addedLines))
	for _, line := range addedLines {
		known[line] = struct{}{}
	}
	for _, line := range removedLines {
		if _, ok := known[line]; !ok {
			lostLines = append(lostLines, line)
		}
	}
	return lostLines
}
