/*
Package htmx provides an embeddable HTML interface to Cuebook documents.

New creates a read-only handler for any fs.FS. NewWithFileSystem adds editing:
changes are applied with cuebook.Commit and written through a
cuebook.FileSystem, such as one from cuebook.NewFileSystem, which replaces
files atomically.

The handler recursively discovers `.cue` files and displays their validated
entries in a navigable file tree. It supports creating entries and editing
fields, rendering full pages for browsers and fragments for HTMX requests.
*/
package htmx

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/dkotik/cuebook"
	gitview "github.com/dkotik/cuebook/htmx/git"
	"github.com/dkotik/cuebook/search"
	"github.com/dkotik/htadaptor"
)

//go:embed templates/*.html
var templateFiles embed.FS

//go:embed assets/*.css assets/*.js assets/*.txt assets/*.svg
var assetFiles embed.FS

type handler struct {
	source        fs.FS
	gitSource     fs.FS
	files         cuebook.FileSystem // nil for read-only handlers
	templates     *template.Template
	searchFS      search.SearchFS
	gitFileLinker gitview.FileLinker
	agentEnabled  bool
	agentBaseURL  string
	routePrefix   string
}

// New returns a read-only HTTP handler for CUE files in source. The filesystem
// is required; opts customize route registration and mounting.
func New(source fs.FS, opts ...Option) (http.Handler, error) {
	return newHandler(source, nil, opts...)
}

// NewWithFileSystem returns an HTTP handler that browses source and saves edits
// to files. source and files must refer to the same logical files; for a
// directory, pass [os.DirFS] and [cuebook.NewFileSystem] of the same root. opts
// configure route registration and mounting.
func NewWithFileSystem(source fs.FS, files cuebook.FileSystem, opts ...Option) (http.Handler, error) {
	if files == nil {
		return nil, ErrFileSystemNil
	}
	return newHandler(source, files, opts...)
}

func newHandler(source fs.FS, files cuebook.FileSystem, opts ...Option) (http.Handler, error) {
	if source == nil {
		return nil, ErrSourceFileSystemNil
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
		return nil, err
	}

	searchableSource, err := search.NewFS(source)
	if err != nil {
		return nil, err
	}
	mux := config.ServeMux
	app := &handler{
		source:        searchableSource,
		gitSource:     source,
		files:         files,
		templates:     templates,
		searchFS:      searchableSource,
		gitFileLinker: config.GitFileLinker,
		agentEnabled:  config.Agent != nil,
		routePrefix:   config.ServeMuxPrefix,
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
		htadaptor.WithQueryValues("file", "folder"),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt list route: %w", err)
	}
	listWorkspaceHandler, err := config.Adaptor.AdaptFunc(app.list,
		htadaptor.WithErrorHandler(errorHandler),
		htadaptor.WithTemplate(templates.Lookup("workspace.html")),
		htadaptor.WithQueryValues("file", "folder"),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt list HTMX route: %w", err)
	}
	mux.Handle("GET "+routeWithPrefix(config.ServeMuxPrefix, "{$}"), NewHTMXSwitch(listPageHandler, listWorkspaceHandler))

	if app.gitFileLinker != nil {
		changesPageHandler, err := config.Adaptor.AdaptFunc(app.changes,
			htadaptor.WithErrorHandler(errorHandler),
			htadaptor.WithTemplate(templates.Lookup("page.html")),
		)
		if err != nil {
			return nil, fmt.Errorf("htmx: adapt Git changes route: %w", err)
		}
		changesWorkspaceHandler, err := config.Adaptor.AdaptFunc(app.changes,
			htadaptor.WithErrorHandler(errorHandler),
			htadaptor.WithTemplate(templates.Lookup("workspace.html")),
		)
		if err != nil {
			return nil, fmt.Errorf("htmx: adapt Git changes HTMX route: %w", err)
		}
		mux.Handle("GET "+routeWithPrefix(config.ServeMuxPrefix, "changes"), NewHTMXSwitch(changesPageHandler, changesWorkspaceHandler))
	}

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

	uuidEditFormHandler, err := config.Adaptor.AdaptFunc(app.editFormUUID,
		htadaptor.WithErrorHandler(errorHandler),
		htadaptor.WithTemplate(templates.Lookup("field-form.html")),
		htadaptor.WithQueryValues("file", "uuid", "field", "mode", "view"),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt UUID edit form route: %w", err)
	}
	uuidEditFieldHandler, err := config.Adaptor.AdaptFunc(app.editFormUUID,
		htadaptor.WithErrorHandler(errorHandler),
		htadaptor.WithTemplate(templates.Lookup("field.html")),
		htadaptor.WithQueryValues("file", "uuid", "field", "mode", "view"),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt UUID edit field route: %w", err)
	}
	mux.Handle("GET "+routeWithPrefix(config.ServeMuxPrefix, "edit/uuid"), selectEditTemplateHandler(
		NewHTMXSwitch(uuidEditFormHandler, uuidEditFormHandler),
		NewHTMXSwitch(uuidEditFieldHandler, uuidEditFieldHandler),
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
		htadaptor.WithTemplate(templates.Lookup("entry-workspace.html")),
		htadaptor.WithQueryValues("path", "file", "head", "tail"),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt entry HTMX route: %w", err)
	}
	mux.Handle("GET "+routeWithPrefix(config.ServeMuxPrefix, "entry"), NewHTMXSwitch(entryPageHandler, entryFragmentHandler))

	uuidEntryPageHandler, err := config.Adaptor.AdaptFunc(app.entryUUID,
		htadaptor.WithErrorHandler(errorHandler),
		htadaptor.WithTemplate(templates.Lookup("page.html")),
		htadaptor.WithQueryValues("path", "file", "uuid"),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt UUID entry route: %w", err)
	}
	uuidEntryFragmentHandler, err := config.Adaptor.AdaptFunc(app.entryUUID,
		htadaptor.WithErrorHandler(errorHandler),
		htadaptor.WithTemplate(templates.Lookup("entry-workspace.html")),
		htadaptor.WithQueryValues("path", "file", "uuid"),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt UUID entry HTMX route: %w", err)
	}
	mux.Handle("GET "+routeWithPrefix(config.ServeMuxPrefix, "entry/uuid"), NewHTMXSwitch(uuidEntryPageHandler, uuidEntryFragmentHandler))

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

	uuidEditPageHandler, err := config.Adaptor.AdaptFunc(app.editUUID,
		htadaptor.WithErrorHandler(editErrorHandler(templates.Lookup("page.html"))),
		htadaptor.WithEncoder(NewEncoder(htadaptor.NewTemplateEncoder(templates.Lookup("page.html")))),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt UUID edit route: %w", err)
	}
	uuidEditWorkspaceHandler, err := config.Adaptor.AdaptFunc(app.editUUID,
		htadaptor.WithErrorHandler(editErrorHandler(templates.Lookup("workspace.html"))),
		htadaptor.WithEncoder(NewEncoder(htadaptor.NewTemplateEncoder(templates.Lookup("workspace.html")))),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt UUID edit HTMX route: %w", err)
	}
	mux.Handle("POST "+routeWithPrefix(config.ServeMuxPrefix, "edit/uuid"), sameOriginMiddleware(NewHTMXSwitch(uuidEditPageHandler, uuidEditWorkspaceHandler)))

	newFieldPageHandler, err := config.Adaptor.AdaptFunc(app.newField,
		htadaptor.WithErrorHandler(editErrorHandler(templates.Lookup("page.html"))),
		htadaptor.WithEncoder(NewEncoder(htadaptor.NewTemplateEncoder(templates.Lookup("page.html")))),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt new field route: %w", err)
	}
	newFieldWorkspaceHandler, err := config.Adaptor.AdaptFunc(app.newField,
		htadaptor.WithErrorHandler(editErrorHandler(templates.Lookup("workspace.html"))),
		htadaptor.WithEncoder(NewEncoder(htadaptor.NewTemplateEncoder(templates.Lookup("workspace.html")))),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt new field HTMX route: %w", err)
	}
	mux.Handle("POST "+routeWithPrefix(config.ServeMuxPrefix, "add-field"), sameOriginMiddleware(NewHTMXSwitch(newFieldPageHandler, newFieldWorkspaceHandler)))

	uuidNewFieldPageHandler, err := config.Adaptor.AdaptFunc(app.newFieldUUID,
		htadaptor.WithErrorHandler(editErrorHandler(templates.Lookup("page.html"))),
		htadaptor.WithEncoder(NewEncoder(htadaptor.NewTemplateEncoder(templates.Lookup("page.html")))),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt UUID new field route: %w", err)
	}
	uuidNewFieldWorkspaceHandler, err := config.Adaptor.AdaptFunc(app.newFieldUUID,
		htadaptor.WithErrorHandler(editErrorHandler(templates.Lookup("workspace.html"))),
		htadaptor.WithEncoder(NewEncoder(htadaptor.NewTemplateEncoder(templates.Lookup("workspace.html")))),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt UUID new field HTMX route: %w", err)
	}
	mux.Handle("POST "+routeWithPrefix(config.ServeMuxPrefix, "add-field/uuid"), sameOriginMiddleware(NewHTMXSwitch(uuidNewFieldPageHandler, uuidNewFieldWorkspaceHandler)))

	formErrorEncoder := NewEncoder(htadaptor.NewTemplateEncoder(templates.Lookup("error.html")))
	formErrorHandler := func(target string) htadaptor.ErrorHandlerFunc {
		return htadaptor.ErrorHandlerFunc(func(w http.ResponseWriter, r *http.Request, cause error) {
			w.Header().Set("HX-Retarget", target)
			w.Header().Set("HX-Reswap", "innerHTML")
			response := pageData{Error: cause.Error()}
			if encodeErr := formErrorEncoder.Encode(w, r, htadaptor.GetHyperTextStatusCode(cause), response); encodeErr != nil {
				panic(encodeErr)
			}
		})
	}

	addPageHandler, err := config.Adaptor.AdaptFunc(app.add,
		htadaptor.WithErrorHandler(errorHandler),
		htadaptor.WithTemplate(templates.Lookup("page.html")),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt add route: %w", err)
	}
	addWorkspaceHandler, err := config.Adaptor.AdaptFunc(app.add,
		htadaptor.WithErrorHandler(formErrorHandler("#add-entry-error")),
		htadaptor.WithTemplate(templates.Lookup("entry-workspace.html")),
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

	uuidMovePageHandler, err := config.Adaptor.AdaptFunc(app.moveUUID,
		htadaptor.WithErrorHandler(errorHandler),
		htadaptor.WithTemplate(templates.Lookup("page.html")),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt UUID move route: %w", err)
	}
	uuidMoveWorkspaceHandler, err := config.Adaptor.AdaptFunc(app.moveUUID,
		htadaptor.WithErrorHandler(errorHandler),
		htadaptor.WithTemplate(templates.Lookup("workspace.html")),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt UUID move HTMX route: %w", err)
	}
	mux.Handle("POST "+routeWithPrefix(config.ServeMuxPrefix, "move/uuid"), sameOriginMiddleware(NewHTMXSwitch(uuidMovePageHandler, uuidMoveWorkspaceHandler)))

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

	uuidArchivePageHandler, err := config.Adaptor.AdaptFunc(app.archiveUUID,
		htadaptor.WithErrorHandler(errorHandler),
		htadaptor.WithTemplate(templates.Lookup("page.html")),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt UUID archive route: %w", err)
	}
	uuidArchiveWorkspaceHandler, err := config.Adaptor.AdaptFunc(app.archiveUUID,
		htadaptor.WithErrorHandler(errorHandler),
		htadaptor.WithTemplate(templates.Lookup("workspace.html")),
	)
	if err != nil {
		return nil, fmt.Errorf("htmx: adapt UUID archive HTMX route: %w", err)
	}
	mux.Handle("POST "+routeWithPrefix(config.ServeMuxPrefix, "archive/uuid"), sameOriginMiddleware(NewHTMXSwitch(uuidArchivePageHandler, uuidArchiveWorkspaceHandler)))

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
