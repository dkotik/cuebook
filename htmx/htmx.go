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
	"github.com/dkotik/htadaptor"
)

//go:embed templates/*.html
var templateFiles embed.FS

//go:embed assets/app.css assets/bulma.css assets/bulma-LICENSE.txt assets/htmx-2.0.4.min.js assets/htmx-LICENSE.txt assets/theme.js assets/file-tree.js assets/entry-move.js assets/entry-move-dialog.js assets/remember-details.js assets/archive-confirm.js assets/move-confirm.js assets/search-control.js assets/live-reload.js assets/entry-paste.js assets/entry-copy.js assets/flash.js assets/favicon.svg
var assetFiles embed.FS

// Committer applies a prepared Cuebook patch to a named source file.
// Implementations should apply patches to fresh source and commit atomically.
type Committer interface {
	Commit(name string, change patch.Patch) error
}

type handler struct {
	source       fs.FS
	committer    Committer
	templates    *template.Template
	searchFS     search.SearchFS
	agentEnabled bool
	agentBaseURL string
	routePrefix  string
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
	}).ParseFS(templateFiles, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("htmx: parse page templates: %w", err)
	}

	searchableSource, err := search.NewFS(source)
	if err != nil {
		return nil, fmt.Errorf("htmx: wrap source filesystem for search: %w", err)
	}
	mux := config.ServeMux
	app := &handler{
		source:       searchableSource,
		committer:    committer,
		templates:    templates,
		searchFS:     searchableSource,
		agentEnabled: config.Agent != nil,
		routePrefix:  config.ServeMuxPrefix,
	}
	if config.Agent != nil {
		app.agentBaseURL = routeWithPrefix(config.ServeMuxPrefix, "agent")
		mountedAgent := config.Agent.Mount(app.agentBaseURL)
		mux.Handle(app.agentBaseURL+"/", http.StripPrefix(app.agentBaseURL, mountedAgent))
	}
	errorHandler := htadaptor.NewErrorHandlerFromTemplate(htadaptor.DefaultErrorTemplate())
	listPageHandler, err := config.Adaptor.AdaptFunc(app.list,
		htadaptor.WithErrorHandler(errorHandler),
		htadaptor.WithTemplate(templates.Lookup("page.html")),
		htadaptor.WithQueryValues("file"),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt list route: %w", err)
	}
	listWorkspaceHandler, err := config.Adaptor.AdaptFunc(app.list,
		htadaptor.WithErrorHandler(errorHandler),
		htadaptor.WithTemplate(templates.Lookup("workspace.html")),
		htadaptor.WithQueryValues("file"),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt list HTMX route: %w", err)
	}
	mux.Handle("GET "+routeWithPrefix(config.ServeMuxPrefix, "{$}"), NewHTMXSwitch(listPageHandler, listWorkspaceHandler))

	editFormHandler, err := config.Adaptor.AdaptFunc(app.editForm,
		htadaptor.WithErrorHandler(errorHandler),
		htadaptor.WithTemplate(templates.Lookup("field-form.html")),
		htadaptor.WithQueryValues("file", "entry", "field", "mode", "view"),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt edit form route: %w", err)
	}
	editFieldHandler, err := config.Adaptor.AdaptFunc(app.editForm,
		htadaptor.WithErrorHandler(errorHandler),
		htadaptor.WithTemplate(templates.Lookup("field.html")),
		htadaptor.WithQueryValues("file", "entry", "field", "mode", "view"),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt edit field route: %w", err)
	}
	mux.Handle("GET "+routeWithPrefix(config.ServeMuxPrefix, "edit"), selectEditTemplateHandler(
		NewHTMXSwitch(editFormHandler, editFormHandler),
		NewHTMXSwitch(editFieldHandler, editFieldHandler),
	))

	entryPageHandler, err := config.Adaptor.AdaptFunc(app.entry,
		htadaptor.WithErrorHandler(errorHandler),
		htadaptor.WithTemplate(templates.Lookup("page.html")),
		htadaptor.WithQueryValues("path", "file", "head", "tail"),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt entry route: %w", err)
	}
	entryFragmentHandler, err := config.Adaptor.AdaptFunc(app.entry,
		htadaptor.WithErrorHandler(errorHandler),
		htadaptor.WithTemplate(templates.Lookup("entry.html")),
		htadaptor.WithQueryValues("path", "file", "head", "tail"),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt entry HTMX route: %w", err)
	}
	mux.Handle("GET "+routeWithPrefix(config.ServeMuxPrefix, "entry"), NewHTMXSwitch(entryPageHandler, entryFragmentHandler))

	searchHandler, err := config.Adaptor.AdaptFunc(app.search,
		htadaptor.WithErrorHandler(errorHandler),
		htadaptor.WithTemplate(templates.Lookup("search-results.html")),
		htadaptor.WithQueryValues("q", "query"),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt search route: %w", err)
	}
	mux.Handle("GET "+routeWithPrefix(config.ServeMuxPrefix, "search"), NewHTMXSwitch(searchHandler, searchHandler))

	mux.Handle(
		"GET "+routeWithPrefix(config.ServeMuxPrefix, "events"),
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
				http.Error(w, err.Error(), http.StatusInternalServerError)
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
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					flusher.Flush()
				}
			}
		}),
	)

	editErrorHandler := func(pageTemplate *template.Template) htadaptor.ErrorHandlerFunc {
		wrappedEncoder := NewEncoder(htadaptor.NewTemplateEncoder(pageTemplate))
		return func(w http.ResponseWriter, r *http.Request, cause error) {
			data, _ := app.loadPage(r.FormValue("file"), cause.Error())
			response := editErrorResponse{pageData: data}
			if encodeErr := wrappedEncoder.Encode(w, r, htadaptor.GetHyperTextStatusCode(cause), response); encodeErr != nil {
				panic(encodeErr)
			}
		}
	}

	editPageHandler, err := config.Adaptor.AdaptFunc(app.edit,
		htadaptor.WithErrorHandler(editErrorHandler(templates.Lookup("page.html"))),
		htadaptor.WithEncoder(NewEncoder(htadaptor.NewTemplateEncoder(templates.Lookup("page.html")))),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt edit route: %w", err)
	}
	editWorkspaceHandler, err := config.Adaptor.AdaptFunc(app.edit,
		htadaptor.WithErrorHandler(editErrorHandler(templates.Lookup("workspace.html"))),
		htadaptor.WithEncoder(NewEncoder(htadaptor.NewTemplateEncoder(templates.Lookup("workspace.html")))),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt edit HTMX route: %w", err)
	}
	mux.Handle("POST "+routeWithPrefix(config.ServeMuxPrefix, "edit"), sameOriginMiddleware(NewHTMXSwitch(editPageHandler, editWorkspaceHandler)))

	addPageHandler, err := config.Adaptor.AdaptFunc(app.add,
		htadaptor.WithErrorHandler(errorHandler),
		htadaptor.WithTemplate(templates.Lookup("page.html")),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt add route: %w", err)
	}
	addWorkspaceHandler, err := config.Adaptor.AdaptFunc(app.add,
		htadaptor.WithErrorHandler(errorHandler),
		htadaptor.WithTemplate(templates.Lookup("workspace.html")),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt add HTMX route: %w", err)
	}
	mux.Handle("POST "+routeWithPrefix(config.ServeMuxPrefix, "add"), sameOriginMiddleware(NewHTMXSwitch(addPageHandler, addWorkspaceHandler)))

	pasteEncoder := NewEncoder(htadaptor.NewTemplateEncoder(templates.Lookup("workspace.html")))
	pasteHandler, err := config.Adaptor.AdaptFunc(app.paste,
		htadaptor.WithEncoder(pasteEncoder),
		htadaptor.WithErrorHandler(htadaptor.ErrorHandlerFunc(func(w http.ResponseWriter, r *http.Request, cause error) {
			response := editErrorResponse{pageData: pageData{Error: cause.Error()}}
			if err := pasteEncoder.Encode(w, r, htadaptor.GetHyperTextStatusCode(cause), response); err != nil {
				panic(err)
			}
		})),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt paste route: %w", err)
	}
	mux.Handle("POST "+routeWithPrefix(config.ServeMuxPrefix, "paste"), sameOriginMiddleware(pasteHandler))

	movePageHandler, err := config.Adaptor.AdaptFunc(app.move,
		htadaptor.WithErrorHandler(errorHandler),
		htadaptor.WithTemplate(templates.Lookup("page.html")),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt move route: %w", err)
	}
	moveWorkspaceHandler, err := config.Adaptor.AdaptFunc(app.move,
		htadaptor.WithErrorHandler(errorHandler),
		htadaptor.WithTemplate(templates.Lookup("workspace.html")),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt move HTMX route: %w", err)
	}
	mux.Handle("POST "+routeWithPrefix(config.ServeMuxPrefix, "move"), sameOriginMiddleware(NewHTMXSwitch(movePageHandler, moveWorkspaceHandler)))

	archivePageHandler, err := config.Adaptor.AdaptFunc(app.archive,
		htadaptor.WithErrorHandler(errorHandler),
		htadaptor.WithTemplate(templates.Lookup("page.html")),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt archive route: %w", err)
	}
	archiveWorkspaceHandler, err := config.Adaptor.AdaptFunc(app.archive,
		htadaptor.WithErrorHandler(errorHandler),
		htadaptor.WithTemplate(templates.Lookup("workspace.html")),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt archive HTMX route: %w", err)
	}
	mux.Handle("POST "+routeWithPrefix(config.ServeMuxPrefix, "archive"), sameOriginMiddleware(NewHTMXSwitch(archivePageHandler, archiveWorkspaceHandler)))

	mux.Handle("GET "+routeWithPrefix(config.ServeMuxPrefix, "assets/{name}"), http.HandlerFunc(serveAsset))

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

	if r.Header.Get("HX-Request") != "" {
		s.HTMX.ServeHTTP(w, r)
		return
	}
	s.Normal.ServeHTTP(w, r)
}
