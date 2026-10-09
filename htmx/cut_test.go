package htmx

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/dkotik/cuebook"
)

func TestCutPasteUsesMoveWorkflow(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		prefix      string
		readOnly    bool
		sameFile    bool
		sourceAfter string
		clipboard   string
		cutFile     string
		cutIndex    string
		incomplete  bool
		constrained bool
		failCommit  bool
		wantStatus  int
		wantNotice  string
		wantSource  []string
		wantTarget  []string
	}{
		{name: "cut relocates instead of duplicating", wantStatus: http.StatusOK, wantSource: []string{"Keep me"}, wantTarget: []string{"Existing", "Move me"}},
		{name: "mounted cut paste", prefix: "/books", wantStatus: http.StatusOK, wantSource: []string{"Keep me"}, wantTarget: []string{"Existing", "Move me"}},
		{name: "same file is a no-op", sameFile: true, wantStatus: http.StatusOK, wantSource: []string{"Keep me", "Move me"}, wantTarget: []string{"Existing"}},
		{name: "external clipboard copy never moves pending cut", clipboard: `{Name: "External copy"}`, wantStatus: http.StatusOK, wantSource: []string{"Keep me", "Move me"}, wantTarget: []string{"Existing", "External copy"}},
		{name: "clipboard line ending conversion still moves", clipboard: "CRLF", wantStatus: http.StatusOK, wantSource: []string{"Keep me"}, wantTarget: []string{"Existing", "Move me"}},
		{name: "changed entry cannot be cut using stale state", sourceAfter: `[{Name: "Keep me"}, {Name: "Changed"}]`, wantStatus: http.StatusNoContent, wantNotice: "changed or moved", wantSource: []string{"Keep me", "Changed"}, wantTarget: []string{"Existing"}},
		{name: "reordered source cannot move wrong entry", sourceAfter: `[{Name: "Move me"}, {Name: "Keep me"}]`, wantStatus: http.StatusNoContent, wantNotice: "changed or moved", wantSource: []string{"Move me", "Keep me"}, wantTarget: []string{"Existing"}},
		{name: "deleted entry cannot be cut again", sourceAfter: `[{Name: "Keep me"}]`, wantStatus: http.StatusNoContent, wantNotice: "no longer exists", wantSource: []string{"Keep me"}, wantTarget: []string{"Existing"}},
		{name: "missing source", cutFile: "missing.cue", wantStatus: http.StatusNoContent, wantNotice: "file not found", wantSource: []string{"Keep me", "Move me"}, wantTarget: []string{"Existing"}},
		{name: "invalid source position", cutIndex: "invalid", wantStatus: http.StatusNoContent, wantNotice: "position is invalid", wantSource: []string{"Keep me", "Move me"}, wantTarget: []string{"Existing"}},
		{name: "incomplete cut state", incomplete: true, wantStatus: http.StatusNoContent, wantNotice: "incomplete", wantSource: []string{"Keep me", "Move me"}, wantTarget: []string{"Existing"}},
		{name: "invalid clipboard is validated only on server", clipboard: "[", wantStatus: http.StatusNoContent, wantNotice: "Unable to parse", wantSource: []string{"Keep me", "Move me"}, wantTarget: []string{"Existing"}},
		{name: "read only cannot cut", readOnly: true, wantStatus: http.StatusNoContent, wantNotice: "read-only", wantSource: []string{"Keep me", "Move me"}, wantTarget: []string{"Existing"}},
		{name: "destination constraints preserve source", constrained: true, wantStatus: http.StatusNoContent, wantNotice: "destination file's CUE constraints", wantSource: []string{"Keep me", "Move me"}, wantTarget: []string{"Existing"}},
		{name: "source commit failure rolls back destination", failCommit: true, wantStatus: http.StatusNoContent, wantNotice: "destination change was rolled back", wantSource: []string{"Keep me", "Move me"}, wantTarget: []string{"Existing"}},
	}
	payloadPattern := regexp.MustCompile(`(?s)<pre id="entry-cue-1" hidden>(.*?)</pre>`)
	fingerprintPattern := regexp.MustCompile(`data-cut-fingerprint="([^"]+)"`)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			const original = `[{Name: "Keep me"}, {Name: "Move me", Note: "Copied details" @cuebook(detail)}]`
			documents := map[string][]byte{"source.cue": []byte(original), "destination.cue": []byte(`[{Name: "Existing"}]`)}
			if tt.constrained {
				documents["destination.cue"] = []byte("#entry: {Name: \"Existing\"}\n[...#entry] & [{Name: \"Existing\"}]")
			}
			files := transferSourceFS(documents)
			committer := &transferCommitter{files: files}
			if tt.failCommit {
				committer.failCalls = map[int]error{2: errors.New("storage failure")}
			}
			var handler http.Handler
			var err error
			if tt.readOnly {
				handler, err = New(files, WithServeMuxPrefix(tt.prefix))
			} else {
				handler, err = NewWithCommitter(files, committer, WithServeMuxPrefix(tt.prefix))
			}
			if err != nil {
				t.Fatal(err)
			}
			book, err := cuebook.New(files["source.cue"].Data)
			if err != nil {
				t.Fatal(err)
			}
			value, err := book.GetValue(1)
			if err != nil {
				t.Fatal(err)
			}
			entryRange, err := cuebook.NewByteRange(value)
			if err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest(http.MethodGet, "http://example.test"+tt.prefix+entryURL("source.cue", entryRange), nil)
			request.Header.Set("HX-Request", "true")
			page := httptest.NewRecorder()
			handler.ServeHTTP(page, request)
			if page.Code != http.StatusOK {
				t.Fatalf("entry status = %d; body: %s", page.Code, page.Body.String())
			}
			payloadMatch := payloadPattern.FindStringSubmatch(page.Body.String())
			if payloadMatch == nil {
				t.Fatal("entry copy payload missing")
			}
			payload := html.UnescapeString(payloadMatch[1])
			fingerprint := fmt.Sprintf("%x", sha256.Sum256([]byte(payload)))
			if tt.readOnly {
				if strings.Contains(page.Body.String(), "data-entry-cut=") {
					t.Fatal("read-only entry has a cut button")
				}
			} else {
				if !strings.Contains(page.Body.String(), `data-entry-cut="entry-cue-1" data-file="source.cue" data-entry-index="1"`) {
					t.Fatal("cut button identifies the wrong source entry")
				}
				match := fingerprintPattern.FindStringSubmatch(page.Body.String())
				if match == nil || match[1] != fingerprint {
					t.Fatal("cut fingerprint does not identify the clipboard payload")
				}
			}
			if tt.sourceAfter != "" {
				waitForSearchResponse(t, handler, "Move")
				files["source.cue"].Data = []byte(tt.sourceAfter)
			}
			beforeSource := bytes.Clone(files["source.cue"].Data)
			beforeTarget := bytes.Clone(files["destination.cue"].Data)
			target := "destination.cue"
			if tt.sameFile {
				target = "source.cue"
			}
			cutFile := "source.cue"
			if tt.cutFile != "" {
				cutFile = tt.cutFile
			}
			cutIndex := "1"
			if tt.cutIndex != "" {
				cutIndex = tt.cutIndex
			}
			clipboard := payload
			if tt.clipboard == "CRLF" {
				clipboard = strings.ReplaceAll(payload, "\n", "\r\n")
			} else if tt.clipboard != "" {
				clipboard = tt.clipboard
			}
			values := url.Values{"file": {target}, "source": {clipboard}, "cut_file": {cutFile}, "cut_entry": {cutIndex}, "cut_fingerprint": {fingerprint}}
			if tt.incomplete {
				values.Del("cut_file")
			}
			response := submitPaste(handler, tt.prefix+"/paste", values)
			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s; flash: %s", response.Code, tt.wantStatus, response.Body.String(), response.Header().Get("HX-Trigger"))
			}
			if tt.wantStatus == http.StatusNoContent {
				if response.Body.Len() != 0 || !strings.Contains(response.Header().Get("HX-Trigger"), tt.wantNotice) {
					t.Errorf("cut failure did not return an empty flash-only response: %q", response.Header().Get("HX-Trigger"))
				}
				if !bytes.Equal(files["source.cue"].Data, beforeSource) || !bytes.Equal(files["destination.cue"].Data, beforeTarget) {
					t.Fatal("failed cut modified a document")
				}
			}
			assertEntryTitles(t, files["source.cue"].Data, tt.wantSource)
			assertEntryTitles(t, files["destination.cue"].Data, tt.wantTarget)
			if tt.failCommit && !equalStrings(committer.calls, []string{"destination.cue", "source.cue", "destination.cue"}) {
				t.Errorf("rollback commit order = %v", committer.calls)
			}
			if tt.wantStatus == http.StatusOK && tt.clipboard != `{Name: "External copy"}` && !tt.sameFile {
				repeated := submitPaste(handler, tt.prefix+"/paste", values)
				if repeated.Code != http.StatusNoContent {
					t.Fatal("replaying a consumed cut moved or duplicated an entry")
				}
				assertEntryTitles(t, files["destination.cue"].Data, tt.wantTarget)
			}
		})
	}
}
