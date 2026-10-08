package htmx

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

func TestDeleteArchivesEntries(t *testing.T) {
	tests := []struct {
		name           string
		initialArchive []byte
		wantArchive    []string
	}{
		{
			name:        "creates daily archive",
			wantArchive: []string{"Move me"},
		},
		{
			name: "appends without replacing existing archive",
			initialArchive: []byte(`#entry: {Name: string @cuebook(title)}
[...#entry] & [{Name: "Already archived"}]
`),
			wantArchive: []string{"Already archived", "Move me"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			files := transferTestDocuments()
			archiveName := archiveDirectory + time.Now().Format("2006-01-02") + ".cue"
			if test.initialArchive != nil {
				files[archiveName] = test.initialArchive
			}
			committer := &archiveTestCommitter{transferCommitter: &transferCommitter{files: transferSourceFS(files)}}
			handler, err := NewWithCommitter(committer.files, committer)
			if err != nil {
				t.Fatal(err)
			}

			pageRequest := httptest.NewRequest(http.MethodGet, "http://example.test/?file=source.cue", nil)
			pageResponse := httptest.NewRecorder()
			handler.ServeHTTP(pageResponse, pageRequest)
			if pageResponse.Code != http.StatusOK {
				t.Fatalf("writable entry page status = %d, want %d; body: %s", pageResponse.Code, http.StatusOK, pageResponse.Body.String())
			}
			if strings.Contains(pageResponse.Body.String(), `action="/delete"`) {
				t.Fatalf("list view unexpectedly shows an archive button: %s", pageResponse.Body.String())
			}
			for _, want := range []string{
				`id="delete-confirmation"`,
				`role="dialog" aria-modal="true"`,
				`aria-labelledby="delete-confirmation-title"`,
				`aria-describedby="delete-confirmation-description"`,
				`<p class="delete-confirmation-eyebrow">Confirm archive</p>`,
				`<script src="/assets/delete-confirm.js" defer></script>`,
			} {
				if !strings.Contains(pageResponse.Body.String(), want) {
					t.Errorf("writable entry page does not contain confirmation markup %q", want)
				}
			}

			response := submitDelete(t, handler, "source.cue", 0, "http://example.test")
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
			}
			if !strings.Contains(response.Body.String(), archiveName) || !strings.Contains(response.Body.String(), "Move me") {
				t.Fatalf("response does not display the archive destination and moved entry: %s", response.Body.String())
			}
			if strings.Contains(response.Body.String(), `action="/delete"`) {
				t.Fatalf("archive entries should not show a delete button: %s", response.Body.String())
			}

			assertEntryTitles(t, committer.files["source.cue"].Data, []string{"Keep me", "Third source"})
			assertEntryTitles(t, committer.files[archiveName].Data, test.wantArchive)
			if want := []string{archiveName, "source.cue"}; !equalStrings(committer.calls, want) {
				t.Fatalf("commit order = %v, want %v", committer.calls, want)
			}
		})
	}
}

func TestDeleteRejectsInvalidRequests(t *testing.T) {
	tests := []struct {
		name       string
		file       string
		entry      int
		readOnly   bool
		noCreator  bool
		origin     string
		wantStatus int
		wantNotice string
	}{
		{
			name:       "read-only source",
			file:       "source.cue",
			readOnly:   true,
			origin:     "http://example.test",
			wantStatus: http.StatusInternalServerError,
			wantNotice: "This source is read-only.",
		},
		{
			name:       "committer cannot create archive files",
			file:       "core1.cue",
			noCreator:  true,
			origin:     "http://example.test",
			wantStatus: http.StatusInternalServerError,
			wantNotice: "This source cannot create archive files.",
		},
		{
			name:       "invalid entry index",
			file:       "source.cue",
			entry:      100,
			origin:     "http://example.test",
			wantStatus: http.StatusNotFound,
			wantNotice: "item not found: path=source.cue",
		},
		{
			name:       "missing source file",
			file:       "missing.cue",
			entry:      0,
			origin:     "http://example.test",
			wantStatus: http.StatusNotFound,
			wantNotice: "file not found: path=missing.cue",
		},
		{
			name:       "entry already in archive",
			file:       archiveDirectory + "old.cue",
			origin:     "http://example.test",
			wantStatus: http.StatusInternalServerError,
			wantNotice: "Entries in the archive cannot be deleted.",
		},
		{
			name:       "cross-origin request",
			file:       "source.cue",
			origin:     "http://attacker.test",
			wantStatus: http.StatusForbidden,
			wantNotice: "Cross-origin edits are not allowed.",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			documents := transferTestDocuments()
			if test.file == archiveDirectory+"old.cue" {
				documents = map[string][]byte{archiveDirectory + "old.cue": documents["source.cue"]}
			}
			files := transferSourceFS(documents)

			var handler http.Handler
			var committer *archiveTestCommitter
			var err error
			switch {
			case test.readOnly:
				handler, err = New(files)
			case test.noCreator:
				handler, err = NewWithCommitter(testSource(), &recordingCommitter{})
			default:
				committer = &archiveTestCommitter{transferCommitter: &transferCommitter{files: files}}
				handler, err = NewWithCommitter(files, committer)
			}
			if err != nil {
				t.Fatal(err)
			}

			response := submitDelete(t, handler, test.file, test.entry, test.origin)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", response.Code, test.wantStatus, response.Body.String())
			}
			if !strings.Contains(response.Body.String(), test.wantNotice) {
				t.Errorf("response missing notice %q: %s", test.wantNotice, response.Body.String())
			}
			if committer != nil && len(committer.calls) != 0 {
				t.Errorf("commit calls = %v, want none", committer.calls)
			}
		})
	}
}

func TestDeleteRollsBackArchiveIfSourceCommitFails(t *testing.T) {
	t.Parallel()

	documents := transferTestDocuments()
	files := transferSourceFS(documents)
	committer := &archiveTestCommitter{transferCommitter: &transferCommitter{
		files:     files,
		failCalls: map[int]error{2: errors.New("source storage failure")},
	}}
	handler, err := NewWithCommitter(files, committer)
	if err != nil {
		t.Fatal(err)
	}
	originalSource := append([]byte(nil), files["source.cue"].Data...)
	archiveName := archiveDirectory + time.Now().Format("2006-01-02") + ".cue"

	response := submitDelete(t, handler, "source.cue", 0, "http://example.test")
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusInternalServerError, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "destination change was rolled back") {
		t.Fatalf("rollback notice missing: %s", response.Body.String())
	}
	if !bytes.Equal(files["source.cue"].Data, originalSource) {
		t.Fatalf("source changed after failed transfer:\n%s", files["source.cue"].Data)
	}
	assertEntryTitles(t, files[archiveName].Data, nil)
	if want := []string{archiveName, "source.cue", archiveName}; !equalStrings(committer.calls, want) {
		t.Fatalf("commit order = %v, want %v", committer.calls, want)
	}
}

func TestDeleteDoesNotFollowArchiveDirectorySymlink(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	outside := t.TempDir()
	source := []byte(`#entry: {Name: string @cuebook(title)}
[...#entry] & [{Name: "Keep in source"}]
`)
	if err := os.WriteFile(filepath.Join(root, "source.cue"), source, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, strings.TrimSuffix(archiveDirectory, "/"))); err != nil {
		t.Fatal(err)
	}
	handler, err := NewDirectory(root)
	if err != nil {
		t.Fatal(err)
	}

	response := submitDelete(t, handler, "source.cue", 0, "http://example.test")
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusInternalServerError, response.Body.String())
	}
	updated, err := os.ReadFile(filepath.Join(root, "source.cue"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(updated, source) {
		t.Fatalf("source changed despite archive creation failure:\n%s", updated)
	}
	entries, err := os.ReadDir(outside)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("archive symlink target unexpectedly contains files: %v", entries)
	}
}

func submitDelete(t *testing.T, handler http.Handler, file string, entry int, origin string) *httptest.ResponseRecorder {
	t.Helper()
	values := url.Values{
		"file":  {file},
		"entry": {strconv.Itoa(entry)},
	}
	request := httptest.NewRequest(http.MethodPost, "http://example.test/delete", strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Origin", origin)
	request.Header.Set("HX-Request", "true")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

type archiveTestCommitter struct {
	*transferCommitter
}

func (c *archiveTestCommitter) CreateFileIfNotExists(name string, content []byte) error {
	if _, exists := c.files[name]; exists {
		return nil
	}
	c.files[name] = &fstest.MapFile{Data: append([]byte(nil), content...)}
	return nil
}

var _ Committer = (*archiveTestCommitter)(nil)
var _ FileCreator = (*archiveTestCommitter)(nil)
