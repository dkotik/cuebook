package git

import (
	"bytes"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
)

func TestParseStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		output  []byte
		want    []changedPath
		wantErr string
	}{
		{
			name:   "clean tree",
			output: nil,
			want:   []changedPath{},
		},
		{
			name:   "modified path with spaces",
			output: []byte(" M path with spaces.txt\x00"),
			want:   []changedPath{{path: "path with spaces.txt", status: "Modified"}},
		},
		{
			name:   "modified path containing a newline",
			output: []byte(" M path with\nnewline.txt\x00"),
			want:   []changedPath{{path: "path with\nnewline.txt", status: "Modified"}},
		},
		{
			name:   "staged and unstaged modification",
			output: []byte("MM edited.txt\x00"),
			want:   []changedPath{{path: "edited.txt", status: "Modified"}},
		},
		{
			name:   "untracked path",
			output: []byte("?? new.txt\x00"),
			want:   []changedPath{{path: "new.txt", status: "Untracked"}},
		},
		{
			name:   "rename links destination only",
			output: []byte("R  new-name.txt\x00old-name.txt\x00"),
			want:   []changedPath{{path: "new-name.txt", status: "Renamed"}},
		},
		{
			name:    "malformed record",
			output:  []byte("M malformed.txt\x00"),
			wantErr: "malformed porcelain status record",
		},
		{
			name:    "changed path escapes working tree",
			output:  []byte(" M ../outside.txt\x00"),
			wantErr: "invalid changed path",
		},
		{
			name:    "rename is missing its source path",
			output:  []byte("R  new-name.txt\x00"),
			wantErr: "missing source path",
		},
		{
			name:    "rename source path escapes working tree",
			output:  []byte("R  new-name.txt\x00../outside.txt\x00"),
			wantErr: "invalid source path",
		},
		{
			name:   "paths are sorted",
			output: []byte(" M z.txt\x00 M a.txt\x00"),
			want: []changedPath{
				{path: "a.txt", status: "Modified"},
				{path: "z.txt", status: "Modified"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseStatus(test.output)
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("parseStatus() error = %v, want it to contain %q", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseStatus() error = %v", err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("parseStatus() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestNewValidatesSourceAndOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		source  fs.FS
		options []Option
		wantErr string
	}{
		{
			name:    "nil filesystem",
			options: []Option{WithFileLinker(testFileLinker{})},
			wantErr: "source filesystem is nil",
		},
		{
			name:    "missing file linker",
			source:  fstest.MapFS{},
			wantErr: "file linker is required",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := New(test.source, test.options...)
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("New() error = %v, want it to contain %q", err, test.wantErr)
			}
		})
	}
}

func TestServeHTTP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		method          string
		wantStatus      int
		wantBody        string
		wantNotInBody   string
		wantContentType string
	}{
		{
			name:            "renders escaped list",
			method:          http.MethodGet,
			wantStatus:      http.StatusOK,
			wantBody:        `&lt;script&gt;.txt</a>`,
			wantNotInBody:   `<script>.txt`,
			wantContentType: "text/html; charset=utf-8",
		},
		{
			name:       "rejects non-get method",
			method:     http.MethodPost,
			wantStatus: http.StatusMethodNotAllowed,
			wantBody:   "Method not allowed.",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			handler := &Handler{changes: []FileChange{{
				Path:   "<script>.txt",
				Status: "Modified",
				Link:   "/files?path=a&other=b",
			}}}
			response := httptest.NewRecorder()
			request := httptest.NewRequest(test.method, "/changes", nil)
			handler.ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
			if !strings.Contains(response.Body.String(), test.wantBody) {
				t.Fatalf("body %q does not contain %q", response.Body.String(), test.wantBody)
			}
			if test.wantNotInBody != "" && strings.Contains(response.Body.String(), test.wantNotInBody) {
				t.Fatalf("body %q unexpectedly contains raw HTML", response.Body.String())
			}
			if got := response.Header().Get("Content-Type"); test.wantContentType != "" && got != test.wantContentType {
				t.Fatalf("Content-Type = %q, want %q", got, test.wantContentType)
			}
			if got := response.Header().Get("X-Content-Type-Options"); test.wantStatus == http.StatusOK && got != "nosniff" {
				t.Fatalf("X-Content-Type-Options = %q, want nosniff", got)
			}
		})
	}
}

func TestServeHTTPRejectsUnsafeLink(t *testing.T) {
	t.Parallel()

	handler := &Handler{changes: []FileChange{{Path: "file.txt", Status: "Modified", Link: "javascript:alert(1)"}}}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/changes", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if bytes.Contains(response.Body.Bytes(), []byte(`href="javascript:alert(1)"`)) {
		t.Fatalf("unsafe link was not filtered: %q", response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte("ZgotmplZ")) {
		t.Fatalf("unsafe link was not replaced by html/template: %q", response.Body.String())
	}
}
