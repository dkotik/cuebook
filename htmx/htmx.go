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
	responseEncoder := adaptorResponseEncoder(templates)

	listHandler, err := config.Adaptor.AdaptFunc(app.list, routeAdaptorOptions(responseEncoder,
		htadaptor.WithTemplate(templates.Lookup("page")),
		htadaptor.WithQueryValues("file"),
		htadaptor.WithMiddleware(templateResponseHeaders(false)),
	)...)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt list route: %w", err)
	}
	listFragmentHandler, err := config.Adaptor.AdaptFunc(app.list, routeAdaptorOptions(responseEncoder,
		htadaptor.WithTemplate(templates.Lookup("workspace")),
		htadaptor.WithQueryValues("file"),
		htadaptor.WithMiddleware(templateResponseHeaders(true)),
	)...)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt list fragment route: %w", err)
	}
	register("GET", "{$}", selectHTMXHandler(listHandler, listFragmentHandler))

	editFormHandler, err := config.Adaptor.AdaptFunc(app.editForm, routeAdaptorOptions(responseEncoder,
		htadaptor.WithTemplate(templates.Lookup("field-form")),
		htadaptor.WithQueryValues("file", "entry", "field", "mode"),
		htadaptor.WithMiddleware(templateResponseHeaders(true)),
	)...)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt edit form route: %w", err)
	}
	editFieldHandler, err := config.Adaptor.AdaptFunc(app.editForm, routeAdaptorOptions(responseEncoder,
		htadaptor.WithTemplate(templates.Lookup("field")),
		htadaptor.WithQueryValues("file", "entry", "field", "mode"),
		htadaptor.WithMiddleware(templateResponseHeaders(true)),
	)...)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt edit field route: %w", err)
	}
	register("GET", "edit", selectEditTemplateHandler(editFormHandler, editFieldHandler))

	itemHandler, err := config.Adaptor.AdaptFunc(app.item, routeAdaptorOptions(responseEncoder,
		htadaptor.WithTemplate(templates.Lookup("entry-item")),
		htadaptor.WithQueryValues("path", "file", "head", "tail"),
		htadaptor.WithMiddleware(templateResponseHeaders(true)),
	)...)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt item route: %w", err)
	}
	register("GET", "item", itemHandler)

	searchHandler, err := config.Adaptor.AdaptFunc(app.search, routeAdaptorOptions(responseEncoder,
		htadaptor.WithTemplate(templates.Lookup("search-results")),
		htadaptor.WithQueryValues("q", "query"),
		htadaptor.WithMiddleware(templateResponseHeaders(true)),
	)...)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt search route: %w", err)
	}
	register("GET", "search", searchHandler)

	clearButtonHandler, err := config.Adaptor.AdaptFunc(app.searchClearButton, routeAdaptorOptions(responseEncoder,
		htadaptor.WithTemplate(templates.Lookup("search-clear-button")),
		htadaptor.WithQueryValues("q"),
		htadaptor.WithMiddleware(templateResponseHeaders(false)),
	)...)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt search clear button route: %w", err)
	}
	register("GET", "search/clear-button", clearButtonHandler)

	eventsHandler, err := config.Adaptor.AdaptFunc(app.liveReloadEvents, routeAdaptorOptions(responseEncoder, htadaptor.WithEncoder(responseEncoder))...)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt live reload route: %w", err)
	}
	register("GET", "events", eventsHandler)

	editHandler, err := adaptPageMutation(config.Adaptor, app.edit, responseEncoder, templates)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt edit route: %w", err)
	}
	register("POST", "edit", editHandler)
	addHandler, err := adaptPageMutation(config.Adaptor, app.add, responseEncoder, templates)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt add route: %w", err)
	}
	register("POST", "add", addHandler)
	moveHandler, err := adaptPageMutation(config.Adaptor, app.move, responseEncoder, templates)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt move route: %w", err)
	}
	register("POST", "move", moveHandler)
	archiveHandler, err := adaptPageMutation(config.Adaptor, app.archive, responseEncoder, templates)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt delete route: %w", err)
	}
	register("POST", "delete", archiveHandler)

	assetHandler, err := config.Adaptor.AdaptFunc(app.asset, routeAdaptorOptions(responseEncoder,
		htadaptor.WithEncoder(responseEncoder),
		htadaptor.WithPathValues("name"),
	)...)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt asset route: %w", err)
	}
	register("GET", "assets/{name}", assetHandler)

	return mux, nil
}

func adaptPageMutation[T any, V htadaptor.Validatable[T], O any](adaptor *htadaptor.Adaptor, call func(context.Context, V) (O, error), errorEncoder htadaptor.Encoder, templates *template.Template) (http.Handler, error) {
	pageHandler, err := adaptor.AdaptFunc(call, formRouteAdaptorOptions(errorEncoder,
		htadaptor.WithTemplate(templates.Lookup("page")),
		htadaptor.WithMiddleware(templateResponseHeaders(false)),
	)...)
	if err != nil {
		return nil, err
	}
	workspaceHandler, err := adaptor.AdaptFunc(call, formRouteAdaptorOptions(errorEncoder,
		htadaptor.WithTemplate(templates.Lookup("workspace")),
		htadaptor.WithMiddleware(templateResponseHeaders(true)),
	)...)
	if err != nil {
		return nil, err
	}
	return selectHTMXHandler(pageHandler, workspaceHandler), nil
}

func selectHTMXHandler(page, workspace http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isHTMX(r) {
			workspace.ServeHTTP(w, r)
			return
		}
		page.ServeHTTP(w, r)
	})
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
