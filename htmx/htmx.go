/*
Package htmx provides an embeddable HTML interface to Cuebook documents.

New creates a read-only handler for any fs.FS. NewWithCommitter adds editing
through an explicit commit implementation, and NewDirectory provides a
filesystem-backed handler that commits edits atomically through patch.Commit.
*/
package htmx

import (
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"net/url"

	"strconv"
	"strings"

	"cuelang.org/go/cue"
	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/metadata"
	"github.com/dkotik/cuebook/patch"
)

//go:embed templates/page.html
var templateFiles embed.FS

//go:embed assets/app.css assets/bulma.css assets/bulma-LICENSE.txt assets/htmx-2.0.4.min.js assets/htmx-LICENSE.txt assets/theme.js assets/file-tree.js
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
// The source and committer must refer to the same logical files.
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
	mux.HandleFunc("POST /edit", app.edit)
	mux.HandleFunc("POST /add", app.add)
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
	case "theme.js", "file-tree.js":
		contentType = "text/javascript; charset=utf-8"
	case "bulma-LICENSE.txt":
		contentType = "text/plain; charset=utf-8"
	case "htmx-2.0.4.min.js":
		contentType = "text/javascript; charset=utf-8"
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

func (a *handler) editForm(w http.ResponseWriter, r *http.Request) {
	if a.committer == nil {
		http.Error(w, "This source is read-only.", http.StatusForbidden)
		return
	}

	fileName := r.URL.Query().Get("file")
	fileNames, err := a.fileNames()
	if err != nil {
		http.Error(w, "Unable to list CUE files.", http.StatusInternalServerError)
		return
	}
	_, document, status, message := a.readDocument(fileName, fileNames)
	if status != http.StatusOK {
		http.Error(w, message, status)
		return
	}

	entryIndex, err := strconv.Atoi(r.URL.Query().Get("entry"))
	if err != nil || entryIndex < 0 {
		http.Error(w, "The entry index is invalid.", http.StatusBadRequest)
		return
	}
	entryValue, err := document.GetValue(entryIndex)
	if err != nil {
		http.Error(w, "Entry not found.", http.StatusNotFound)
		return
	}
	entry, err := cuebook.NewEntry(entryValue)
	if err != nil {
		http.Error(w, "Unable to read this entry.", http.StatusUnprocessableEntity)
		return
	}
	fieldName := r.URL.Query().Get("field")
	field, ok := entry.GetFieldByName(fieldName)
	if !ok || fieldName == "" {
		http.Error(w, "Field not found.", http.StatusNotFound)
		return
	}

	view := makeFieldView(field, fileName, entryIndex, false)
	templateName := "field-form"
	if r.URL.Query().Get("mode") == "view" {
		templateName = "field"
	}
	a.renderFieldTemplate(w, r, templateName, view)
}

func (a *handler) renderFieldTemplate(w http.ResponseWriter, r *http.Request, name string, view fieldView) {
	var output strings.Builder
	if err := a.templates.ExecuteTemplate(&output, name, view); err != nil {
		http.Error(w, "Unable to render the field.", http.StatusInternalServerError)
		return
	}
	w.Header().Add("Vary", "HX-Request")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(output.String()))
}

func (a *handler) edit(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(r) {
		a.renderPage(w, r, pageData{ReadOnly: a.committer == nil, Error: "Cross-origin edits are not allowed."}, http.StatusForbidden)
		return
	}
	if a.committer == nil {
		a.editFailure(w, r, "", "This source is read-only.", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := r.ParseForm(); err != nil {
		a.editFailure(w, r, "", "The edit request is invalid.", http.StatusBadRequest)
		return
	}

	fileName := r.PostForm.Get("file")
	fileNames, err := a.fileNames()
	if err != nil {
		a.editFailure(w, r, "", "Unable to list CUE files.", http.StatusInternalServerError)
		return
	}
	raw, document, status, message := a.readDocument(fileName, fileNames)
	if status != http.StatusOK {
		a.editFailure(w, r, fileName, message, status)
		return
	}
	page, _ := a.pageForDocument(fileName, fileNames, document, "")

	entryIndex, err := strconv.Atoi(r.PostForm.Get("entry"))
	if err != nil || entryIndex < 0 {
		a.renderPage(w, r, withNotice(page, "The entry index is invalid."), http.StatusBadRequest)
		return
	}
	entryValue, err := document.GetValue(entryIndex)
	if err != nil {
		a.renderPage(w, r, withNotice(page, "Entry not found."), http.StatusNotFound)
		return
	}
	entry, err := cuebook.NewEntry(entryValue)
	if err != nil {
		a.renderPage(w, r, withNotice(page, "Unable to read this entry."), http.StatusUnprocessableEntity)
		return
	}
	fieldName := r.PostForm.Get("field")
	field, ok := entry.GetFieldByName(fieldName)
	if !ok || fieldName == "" {
		a.renderPage(w, r, withNotice(page, "Field not found."), http.StatusNotFound)
		return
	}
	if !r.PostForm.Has("value") {
		a.renderPage(w, r, withNotice(page, "The field value is missing."), http.StatusBadRequest)
		return
	}
	value := r.PostForm.Get("value")
	if isSecretField(field.Value) && value == "" && field.String() != "" {
		a.finishEdit(w, r, fileName)
		return
	}

	change, err := patch.UpdateFieldValue(raw, entryValue, field.Value, value)
	if err != nil {
		a.renderPage(w, r, withNotice(page, "The field value could not be formatted."), http.StatusUnprocessableEntity)
		return
	}
	candidate, err := change.ApplyToCueSource(raw)
	if err != nil {
		a.renderPage(w, r, withNotice(page, "The entry changed before the edit could be applied. Reload and try again."), http.StatusConflict)
		return
	}
	if _, err = cuebook.New(candidate); err != nil {
		a.renderPage(w, r, withNotice(page, "The submitted value does not satisfy the CUE constraints: "+err.Error()), http.StatusUnprocessableEntity)
		return
	}
	if err = a.committer.Commit(fileName, change); err != nil {
		status = http.StatusInternalServerError
		notice := "The edit could not be saved."
		if errors.Is(err, patch.ErrByteRangeNotFound) {
			status = http.StatusConflict
			notice = "The entry changed before the edit could be applied. Reload and try again."
		}
		a.editFailure(w, r, fileName, notice, status)
		return
	}
	a.finishEdit(w, r, fileName)
}

func (a *handler) finishEdit(w http.ResponseWriter, r *http.Request, fileName string) {
	if !isHTMX(r) {
		http.Redirect(w, r, fileURL(fileName), http.StatusSeeOther)
		return
	}
	data, status := a.loadPage(fileName, "")
	if status != http.StatusOK {
		data.Error = "The edit was saved, but the updated document could not be displayed."
		status = http.StatusInternalServerError
	}

	a.renderPage(w, r, data, status)
}

func (a *handler) editFailure(w http.ResponseWriter, r *http.Request, fileName, notice string, status int) {
	data, _ := a.loadPage(fileName, notice)
	a.renderPage(w, r, data, status)
}

func sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return !strings.EqualFold(r.Header.Get("Sec-Fetch-Site"), "cross-site")
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return false
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	} else if r.URL.Scheme != "" {
		scheme = r.URL.Scheme
	}
	return strings.EqualFold(parsed.Host, r.Host) && strings.EqualFold(parsed.Scheme, scheme)
}

func isSecretField(field cue.Value) bool {
	_, ok := metadata.GetFieldAttributes(field, "cuebook").GetFirstOf("argon2id")
	return ok
}

func withNotice(data pageData, notice string) pageData {
	data.Error = notice
	return data
}
