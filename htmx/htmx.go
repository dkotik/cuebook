/*
Package htmx provides an embeddable HTML interface to Cuebook documents.

New creates a read-only handler for any fs.FS. NewWithCommitter adds editing
through an explicit commit implementation, and NewDirectory provides a
filesystem-backed handler that commits edits atomically through patch.Commit.

The handler recursively discovers `.cue` files and displays their validated
entries in a navigable file tree. It supports creating entries and editing
fields, rendering full pages for browsers and fragments for HTMX requests.
`New` provides read-only browsing; `NewWithCommitter` and `NewDirectory` enable
writes through the patch workflow.
*/
package htmx

import (
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strings"

	"github.com/dkotik/cuebook/patch"
)

//go:embed templates/page.html
var templateFiles embed.FS

//go:embed assets/app.css assets/bulma.css assets/bulma-LICENSE.txt assets/htmx-2.0.4.min.js assets/htmx-LICENSE.txt assets/theme.js assets/file-tree.js assets/entry-move.js assets/remember-details.js assets/delete-confirm.js assets/move-confirm.js assets/favicon.svg
var assetFiles embed.FS

// Committer applies a prepared Cuebook patch to a named source file.
// Implementations should apply patches to fresh source and commit atomically.
type Committer interface {
	Commit(name string, change patch.Patch) error
}

type handler struct {
	source    fs.FS
	committer Committer
	templates *template.Template
}

// New returns a read-only HTTP handler for CUE files in source.
func New(source fs.FS) (http.Handler, error) {
	return newHandler(source, nil)
}

// NewWithCommitter returns an HTTP handler that can save edits using committer.
// The source and committer must refer to the same logical files. To enable
// archiving entries, committer must also implement FileCreator.
func NewWithCommitter(source fs.FS, committer Committer) (http.Handler, error) {
	if committer == nil {
		return nil, errors.New("htmx: committer is nil")
	}
	return newHandler(source, committer)
}

func newHandler(source fs.FS, committer Committer) (http.Handler, error) {
	if source == nil {
		return nil, errors.New("htmx: source filesystem is nil")
	}
	templates, err := template.ParseFS(templateFiles, "templates/page.html")
	if err != nil {
		return nil, fmt.Errorf("htmx: parse page templates: %w", err)
	}

	app := &handler{
		source:    source,
		committer: committer,
		templates: templates,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", app.index)
	mux.HandleFunc("GET /edit", app.editForm)
	mux.HandleFunc("GET /item", app.item)
	mux.HandleFunc("POST /edit", app.edit)
	mux.HandleFunc("POST /add", app.add)
	mux.HandleFunc("POST /move", app.move)
	mux.HandleFunc("POST /delete", app.archive)
	mux.HandleFunc("GET /assets/{name}", app.asset)
	return securityHeaders(mux), nil
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; base-uri 'none'; frame-ancestors 'none'; form-action 'self'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "same-origin")
		next.ServeHTTP(w, r)
	})
}

func (a *handler) asset(w http.ResponseWriter, r *http.Request) {
	var (
		contentType string
		name        = r.PathValue("name")
	)
	switch name {
	case "app.css", "bulma.css":
		contentType = "text/css; charset=utf-8"
	case "theme.js", "file-tree.js", "entry-move.js", "remember-details.js", "delete-confirm.js", "move-confirm.js":
		contentType = "text/javascript; charset=utf-8"
	case "bulma-LICENSE.txt":
		contentType = "text/plain; charset=utf-8"
	case "htmx-2.0.4.min.js":
		contentType = "text/javascript; charset=utf-8"
	case "favicon.svg":
		contentType = "image/svg+xml"
	case "htmx-LICENSE.txt":
		contentType = "text/plain; charset=utf-8"
	default:
		http.NotFound(w, r)
		return
	}
	content, err := assetFiles.ReadFile("assets/" + name)
	if err != nil {
		http.Error(w, "asset unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = w.Write(content)
}

func isHTMX(r *http.Request) bool {
	return strings.EqualFold(strings.TrimSpace(r.Header.Get("HX-Request")), "true")
}

func (a *handler) renderPage(w http.ResponseWriter, r *http.Request, data pageData, status int) {
	data.RequiredAddFields, data.OptionalAddFields = splitAddFieldViews(data.AddFields)
	var output strings.Builder
	name := "page"
	if isHTMX(r) {
		name = "workspace"
		w.Header().Add("Vary", "HX-Request")
	}
	if err := a.templates.ExecuteTemplate(&output, name, data); err != nil {
		http.Error(w, "Unable to render the page.", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(output.String()))
}
