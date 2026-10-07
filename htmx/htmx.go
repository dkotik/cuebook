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
	"context"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strings"

	"github.com/dkotik/cuebook/patch"
	"github.com/dkotik/cuebook/search"
	"github.com/dkotik/htadaptor"
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
	register := func(method, route string, endpoint http.Handler) {
		mux.Handle(method+" "+routeWithPrefix(config.ServeMuxPrefix, route), endpoint)
	}
	listPageHandler, err := config.Adaptor.AdaptFunc(app.list,
		htadaptor.WithErrorHandler(responseErrorHandler(htadaptor.NewTemplateEncoder(templates.Lookup("page")))),
		htadaptor.WithTemplate(templates.Lookup("page")),
		htadaptor.WithQueryValues("file"),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt list route: %w", err)
	}
	listWorkspaceHandler, err := config.Adaptor.AdaptFunc(app.list,
		htadaptor.WithErrorHandler(responseErrorHandler(htadaptor.NewTemplateEncoder(templates.Lookup("workspace")))),
		htadaptor.WithTemplate(templates.Lookup("workspace")),
		htadaptor.WithQueryValues("file"),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt list HTMX route: %w", err)
	}
	register("GET", "{$}", NewHTMXSwitch(listPageHandler, listWorkspaceHandler))

	editFormHandler, err := config.Adaptor.AdaptFunc(app.editForm,
		htadaptor.WithErrorHandler(responseErrorHandler(htadaptor.NewTemplateEncoder(templates.Lookup("field-form")))),
		htadaptor.WithTemplate(templates.Lookup("field-form")),
		htadaptor.WithQueryValues("file", "entry", "field", "mode"),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt edit form route: %w", err)
	}
	editFieldHandler, err := config.Adaptor.AdaptFunc(app.editForm,
		htadaptor.WithErrorHandler(responseErrorHandler(htadaptor.NewTemplateEncoder(templates.Lookup("field")))),
		htadaptor.WithTemplate(templates.Lookup("field")),
		htadaptor.WithQueryValues("file", "entry", "field", "mode"),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt edit field route: %w", err)
	}
	register("GET", "edit", selectEditTemplateHandler(
		NewHTMXSwitch(editFormHandler, editFormHandler),
		NewHTMXSwitch(editFieldHandler, editFieldHandler),
	))

	itemPageHandler, err := config.Adaptor.AdaptFunc(app.item,
		htadaptor.WithErrorHandler(responseErrorHandler(htadaptor.NewTemplateEncoder(templates.Lookup("page")))),
		htadaptor.WithTemplate(templates.Lookup("page")),
		htadaptor.WithQueryValues("path", "file", "head", "tail"),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt item route: %w", err)
	}
	itemFragmentHandler, err := config.Adaptor.AdaptFunc(app.item,
		htadaptor.WithErrorHandler(responseErrorHandler(htadaptor.NewTemplateEncoder(templates.Lookup("entry-item")))),
		htadaptor.WithTemplate(templates.Lookup("entry-item")),
		htadaptor.WithQueryValues("path", "file", "head", "tail"),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt item HTMX route: %w", err)
	}
	register("GET", "item", NewHTMXSwitch(itemPageHandler, itemFragmentHandler))

	searchHandler, err := config.Adaptor.AdaptFunc(app.search,
		htadaptor.WithErrorHandler(responseErrorHandler(htadaptor.NewTemplateEncoder(templates.Lookup("search-results")))),
		htadaptor.WithTemplate(templates.Lookup("search-results")),
		htadaptor.WithQueryValues("q", "query"),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt search route: %w", err)
	}
	register("GET", "search", NewHTMXSwitch(searchHandler, searchHandler))

	clearButtonHandler, err := config.Adaptor.AdaptFunc(app.searchClearButton,
		htadaptor.WithErrorHandler(responseErrorHandler(htadaptor.NewTemplateEncoder(templates.Lookup("search-clear-button")))),
		htadaptor.WithTemplate(templates.Lookup("search-clear-button")),
		htadaptor.WithQueryValues("q"),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt search clear button route: %w", err)
	}
	register("GET", "search/clear-button", NewHTMXSwitch(clearButtonHandler, clearButtonHandler))

	eventsHandler := http.HandlerFunc(serveLiveReloadEvents)
	register("GET", "events", NewHTMXSwitch(eventsHandler, eventsHandler))

	editPageHandler, err := config.Adaptor.AdaptFunc(app.edit,
		htadaptor.WithErrorHandler(responseErrorHandler(htadaptor.NewTemplateEncoder(templates.Lookup("page")))),
		htadaptor.WithTemplate(templates.Lookup("page")),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt edit route: %w", err)
	}
	editWorkspaceHandler, err := config.Adaptor.AdaptFunc(app.edit,
		htadaptor.WithErrorHandler(responseErrorHandler(htadaptor.NewTemplateEncoder(templates.Lookup("workspace")))),
		htadaptor.WithTemplate(templates.Lookup("workspace")),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt edit HTMX route: %w", err)
	}
	register("POST", "edit", NewHTMXSwitch(editPageHandler, editWorkspaceHandler))

	addPageHandler, err := config.Adaptor.AdaptFunc(app.add,
		htadaptor.WithErrorHandler(responseErrorHandler(htadaptor.NewTemplateEncoder(templates.Lookup("page")))),
		htadaptor.WithTemplate(templates.Lookup("page")),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt add route: %w", err)
	}
	addWorkspaceHandler, err := config.Adaptor.AdaptFunc(app.add,
		htadaptor.WithErrorHandler(responseErrorHandler(htadaptor.NewTemplateEncoder(templates.Lookup("workspace")))),
		htadaptor.WithTemplate(templates.Lookup("workspace")),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt add HTMX route: %w", err)
	}
	register("POST", "add", NewHTMXSwitch(addPageHandler, addWorkspaceHandler))

	movePageHandler, err := config.Adaptor.AdaptFunc(app.move,
		htadaptor.WithErrorHandler(responseErrorHandler(htadaptor.NewTemplateEncoder(templates.Lookup("page")))),
		htadaptor.WithTemplate(templates.Lookup("page")),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt move route: %w", err)
	}
	moveWorkspaceHandler, err := config.Adaptor.AdaptFunc(app.move,
		htadaptor.WithErrorHandler(responseErrorHandler(htadaptor.NewTemplateEncoder(templates.Lookup("workspace")))),
		htadaptor.WithTemplate(templates.Lookup("workspace")),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt move HTMX route: %w", err)
	}
	register("POST", "move", NewHTMXSwitch(movePageHandler, moveWorkspaceHandler))

	archivePageHandler, err := config.Adaptor.AdaptFunc(app.archive,
		htadaptor.WithErrorHandler(responseErrorHandler(htadaptor.NewTemplateEncoder(templates.Lookup("page")))),
		htadaptor.WithTemplate(templates.Lookup("page")),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt delete route: %w", err)
	}
	archiveWorkspaceHandler, err := config.Adaptor.AdaptFunc(app.archive,
		htadaptor.WithErrorHandler(responseErrorHandler(htadaptor.NewTemplateEncoder(templates.Lookup("workspace")))),
		htadaptor.WithTemplate(templates.Lookup("workspace")),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt delete HTMX route: %w", err)
	}
	register("POST", "delete", NewHTMXSwitch(archivePageHandler, archiveWorkspaceHandler))

	register("GET", "assets/{name}", http.HandlerFunc(serveAsset))

	return mux, nil
}

func selectEditTemplateHandler(form, view http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("mode") == "view" {
			view.ServeHTTP(w, r)
			return
		}
		form.ServeHTTP(w, r)
	})
}

func (a *handler) route(target string) string {
	return routeWithPrefix(a.routePrefix, target)
}

func isHTMX(r *http.Request) bool {
	return strings.EqualFold(strings.TrimSpace(r.Header.Get("HX-Request")), "true")
}

type htmxSwitch struct {
	Normal http.Handler
	HTMX   http.Handler
}

func NewHTMXSwitch(normal, htmx http.Handler) http.Handler {
	return htmxSwitch{
		Normal: normal,
		HTMX:   htmx,
	}
}

func (s htmxSwitch) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Vary", "HX-Request")

	ctx := context.WithValue(r.Context(), htmxContextFlag, isHTMX(r))
	ctx = context.WithValue(ctx, sameOriginContextFlag, sameOrigin(r))
	r = r.WithContext(ctx)

	if r.Header.Get("HX-Request") != "" {
		s.HTMX.ServeHTTP(w, r)
		return
	}
	s.Normal.ServeHTTP(w, r)
}
