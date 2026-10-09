package agent

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

type contextFile struct {
	path    string
	format  string
	content string
}

type contextIndex struct {
	files []contextFile
}

func loadContext(source fs.FS, configured options) (contextIndex, error) {
	index := contextIndex{}
	var indexedBytes int64
	err := fs.WalkDir(source, ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("agent: walk context filesystem at %q: %w", name, walkErr)
		}
		if name == "." || entry.IsDir() || !fs.ValidPath(name) {
			return nil
		}
		format, supported := fileFormat(name)
		if !supported || entry.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("agent: inspect context file %q: %w", name, err)
		}
		if !info.Mode().IsRegular() || info.Size() < 0 || info.Size() > configured.maxFileBytes {
			return nil
		}
		if info.Size() > configured.maxIndexedBytes-indexedBytes {
			return nil
		}
		file, err := source.Open(name)
		if err != nil {
			return fmt.Errorf("agent: open context file %q: %w", name, err)
		}
		content, readErr := io.ReadAll(io.LimitReader(file, configured.maxFileBytes+1))
		closeErr := file.Close()
		if readErr != nil || closeErr != nil {
			return fmt.Errorf("agent: read context file %q: %w", name, errors.Join(readErr, closeErr))
		}
		if int64(len(content)) > configured.maxFileBytes || int64(len(content)) > configured.maxIndexedBytes-indexedBytes {
			return nil
		}
		indexedBytes += int64(len(content))
		index.files = append(index.files, contextFile{
			path:    name,
			format:  format,
			content: strings.ToValidUTF8(string(content), "�"),
		})
		return nil
	})
	if err != nil {
		return contextIndex{}, err
	}
	return index, nil
}

func fileFormat(name string) (string, bool) {
	switch strings.ToLower(path.Ext(name)) {
	case ".cue":
		return "CUE", true
	case ".toml":
		return "TOML", true
	case ".json":
		return "JSON", true
	case ".yaml", ".yml":
		return "YAML", true
	case ".md", ".markdown":
		return "Markdown", true
	case ".txt":
		return "text", true
	default:
		return "", false
	}
}

func (index contextIndex) selectFor(question string, configured options) ([]contextFile, []string) {
	if len(index.files) == 0 || configured.maxContextBytes == 0 {
		return nil, nil
	}
	terms := queryTerms(question)
	type scoredFile struct {
		file  contextFile
		score int
	}
	scored := make([]scoredFile, 0, len(index.files))
	for _, file := range index.files {
		score := 0
		for _, term := range terms {
			score += strings.Count(strings.ToLower(file.content), term)
			score += 4 * strings.Count(strings.ToLower(file.path), term)
		}
		scored = append(scored, scoredFile{file: file, score: score})
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		return scored[i].file.path < scored[j].file.path
	})

	selected := make([]contextFile, 0, configured.maxContextFiles)
	sources := make([]string, 0, configured.maxContextFiles)
	usedBytes := 0
	for _, candidate := range scored {
		if len(selected) >= configured.maxContextFiles || usedBytes >= configured.maxContextBytes {
			break
		}
		file := candidate.file
		header := fmt.Sprintf("\n--- source: %s (%s) ---\n", file.path, file.format)
		remaining := configured.maxContextBytes - usedBytes - len(header)
		if remaining <= 0 {
			break
		}
		content := prefixUTF8(file.content, remaining)
		if content == "" {
			continue
		}
		file.content = header + content
		selected = append(selected, file)
		sources = append(sources, candidate.file.path)
		usedBytes += len(file.content)
	}
	return selected, sources
}

func queryTerms(question string) []string {
	seen := make(map[string]struct{})
	terms := make([]string, 0)
	var current strings.Builder
	flush := func() {
		term := strings.ToLower(current.String())
		current.Reset()
		if len([]rune(term)) < 2 {
			return
		}
		if _, exists := seen[term]; exists {
			return
		}
		seen[term] = struct{}{}
		terms = append(terms, term)
	}
	for _, char := range question {
		if unicode.IsLetter(char) || unicode.IsDigit(char) || char == '_' {
			current.WriteRune(char)
			continue
		}
		flush()
	}
	flush()
	return terms
}

func prefixUTF8(value string, maxBytes int) string {
	if len(value) <= maxBytes {
		return value
	}
	end := maxBytes
	for end > 0 && !utf8.RuneStart(value[end]) {
		end--
	}
	return value[:end]
}
