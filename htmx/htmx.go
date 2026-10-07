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
	"time"

	"github.com/dkotik/cuebook/patch"
	"github.com/dkotik/cuebook/search"
)

//go:embed templates/page.html
var templateFiles embed.FS

//go:embed assets/app.css assets/bulma.css assets/bulma-LICENSE.txt assets/htmx-2.0.4.min.js assets/htmx-LICENSE.txt assets/theme.js assets/file-tree.js assets/entry-move.js assets/remember-details.js assets/delete-confirm.js assets/move-confirm.js assets/live-reload.js assets/favicon.svg
var assetFiles embed.FS

// Committer applies a prepared Cuebook patch to a named source file.
// Implementations should apply patches to fresh source and commit atomically.
type Committer interface {
	Commit(name string, change patch.Patch) error
}

type handler struct {
	source      fs.FS
	committer   Committer
	templates   *template.Template
	searchFS    search.SearchFS
	routePrefix string
}

// New returns a read-only HTTP handler for CUE files in source. The filesystem
// is required; opts customize route registration and mounting.
func New(source fs.FS, opts ...Option) (http.Handler, error) {
	return newHandler(source, nil, opts...)
}

// NewWithCommitter returns an HTTP handler that can save edits using committer.
// The source and committer must refer to the same logical files. To enable
// archiving entries, committer must also implement FileCreator. opts configure
// route registration and mounting.
func NewWithCommitter(source fs.FS, committer Committer, opts ...Option) (http.Handler, error) {
	if committer == nil {
		return nil, errors.New("htmx: committer is nil")
	}
	return newHandler(source, committer, opts...)
}

func newHandler(source fs.FS, committer Committer, opts ...Option) (http.Handler, error) {
	if source == nil {
		return nil, errors.New("htmx: source filesystem is nil")
	}
	config, err := resolveOptions(opts)
	if err != nil {
		return nil, err
	}
	templates, err := template.New("page.html").Funcs(template.FuncMap{
		"route": func(target string) string {
			return routeWithPrefix(config.ServeMuxPrefix, target)
		},
	}).ParseFS(templateFiles, "templates/page.html")
	if err != nil {
		return nil, fmt.Errorf("htmx: parse page templates: %w", err)
	}

	searchableSource, err := search.NewFS(source)
	if err != nil {
		return nil, fmt.Errorf("htmx: wrap source filesystem for search: %w", err)
	}
	app := &handler{
		source:      searchableSource,
		committer:   committer,
		templates:   templates,
		searchFS:    searchableSource,
		routePrefix: config.ServeMuxPrefix,
	}
	mux := config.ServeMux
	usingCustomMux := mux != nil
	register := func(method, route string, endpoint http.HandlerFunc) {
		var handler http.Handler = endpoint
		mux.Handle(method+" "+routeWithPrefix(config.ServeMuxPrefix, route), handler)
	}
	register("GET", "{$}", app.list)
	register("GET", "edit", app.editForm)
	register("GET", "item", app.item)
	register("GET", "search", app.search)
	register("GET", "search/clear-button", app.searchClearButton)
	register("GET", "events", app.liveReloadEvents)
	register("POST", "edit", app.edit)
	register("POST", "add", app.add)
	register("POST", "move", app.move)
	register("POST", "delete", app.archive)
	register("GET", "assets/{name}", app.asset)
	if usingCustomMux {
		return mux, nil
	}
	return mux, nil
}

func (a *handler) asset(w http.ResponseWriter, r *http.Request) {
	var (
		contentType string
		name        = r.PathValue("name")
	)
	switch name {
	case "app.css", "bulma.css":
		contentType = "text/css; charset=utf-8"
	case "theme.js", "file-tree.js", "entry-move.js", "remember-details.js", "delete-confirm.js", "move-confirm.js", "live-reload.js":
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

func (a *handler) liveReloadEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE streaming is unavailable.", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	if _, err := fmt.Fprint(w, ": connected\n\n"); err != nil {
		return
	}
	flusher.Flush()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			if _, err := fmt.Fprint(w, ": keep-alive\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func (a *handler) route(target string) string {
	return routeWithPrefix(a.routePrefix, target)
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
