package htmx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReadOnlyHandler(t *testing.T) {
	handler, err := New(testSource())
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		method     string
		path       string
		htmx       bool
		wantStatus int
		contains   []string
		omits      []string
	}{
		{
			name:       "index defaults to dark and offers a theme toggle",
			method:     http.MethodGet,
			path:       "/",
			wantStatus: http.StatusOK,
			contains:   []string{"core1.cue", "subfolder", "sub1.cue", "file=subfolder%2Fsub1.cue", `<file-tree-node class="file-tree-node" data-node-path="subfolder">`, `class="tree-folder-icon"`, `class="tree-file-icon"`, `class="tree-children"`, "bulma.css", "htmx-2.0.4.min.js", "/assets/theme.js", "/assets/file-tree.js", `data-theme="dark"`, `id="theme-toggle"`, `aria-pressed="true"`, `data-theme-icon="moon" style="display: inline-block"`, `d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"`, `data-theme-icon="sun" style="display: none"`, `<circle cx="12" cy="12" r="4"/>`},
			omits:      []string{"notes.txt", "First11111aa", "Dark mode", "Light mode"},
		},
		{
			name:       "selected nested file renders entries",
			method:     http.MethodGet,
			path:       "/?file=subfolder%2Fsub1.cue",
			wantStatus: http.StatusOK,
			contains:   []string{"subfolder/sub1.cue", "First11111aa", "test1@testdomain.com", "Read-only source", `aria-current="page"`},
			omits:      []string{`hx-post="/edit"`, `hx-post="/add"`},
		},
		{
			name:       "htmx request gets workspace fragment",
			method:     http.MethodGet,
			path:       "/?file=core1.cue",
			htmx:       true,
			wantStatus: http.StatusOK,
			contains:   []string{`<main id="workspace"`, "First11111aa"},
			omits:      []string{"<!doctype html>", "file-nav"},
		},
		{
			name:       "path traversal is not found",
			method:     http.MethodGet,
			path:       "/?file=..%2Fsecret.cue",
			wantStatus: http.StatusNotFound,
			contains:   []string{"CUE file not found."},
		},
		{
			name:       "missing file is not found",
			method:     http.MethodGet,
			path:       "/?file=missing.cue",
			wantStatus: http.StatusNotFound,
			contains:   []string{"CUE file not found."},
		},
		{
			name:       "invalid cue is reported safely",
			method:     http.MethodGet,
			path:       "/?file=invalid.cue",
			wantStatus: http.StatusUnprocessableEntity,
			contains:   []string{"Unable to parse or validate this CUE document"},
		},
		{
			name:       "untrusted cue content is escaped",
			method:     http.MethodGet,
			path:       "/?file=escaped.cue",
			wantStatus: http.StatusOK,
			contains:   []string{`&lt;script&gt;alert(1)&lt;/script&gt;`},
			omits:      []string{`<script>alert(1)</script>`},
		},
		{
			name:       "unsupported method is rejected",
			method:     http.MethodPut,
			path:       "/",
			wantStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(tt.method, "http://example.test"+tt.path, nil)
			if tt.htmx {
				request.Header.Set("HX-Request", "true")
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", response.Code, tt.wantStatus, response.Body.String())
			}
			body := response.Body.String()
			for _, want := range tt.contains {
				if !strings.Contains(body, want) {
					t.Errorf("body does not contain %q", want)
				}
			}
			for _, unwanted := range tt.omits {
				if strings.Contains(body, unwanted) {
					t.Errorf("body unexpectedly contains %q", unwanted)
				}
			}
		})
	}
}
