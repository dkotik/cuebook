package htmx

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"
)

type listResponse struct {
	pageData
	statusCode int
}

type itemResponse struct {
	entryView
	statusCode int
	message    string
}

type editFormResponse struct {
	fieldView
	templateName string
	statusCode   int
	message      string
}

type searchResponse struct {
	searchResultsView
	statusCode int
	message    string
	retryAfter string
}

type searchClearButtonResponse struct {
	Visible    bool
	statusCode int
}

type liveReloadResponse struct {
	statusCode int
}

type editResponse struct {
	pageData
	statusCode  int
	redirectURL string
}

type addResponse struct {
	pageData
	statusCode  int
	redirectURL string
}

type moveResponse struct {
	pageData
	statusCode  int
	redirectURL string
}

type archiveResponse struct {
	pageData
	statusCode  int
	redirectURL string
}

type assetResponse struct {
	statusCode  int
	contentType string
	body        []byte
	message     string
}

type responseFailure struct {
	statusCode int
	value      any
}

func (e *responseFailure) Error() string {
	return "domain response requires a non-success HTTP status"
}

func (e *responseFailure) HyperTextStatusCode() int {
	return e.statusCode
}

func responseResult[T any](value T, status int) (T, error) {
	if status == 0 || status == http.StatusOK {
		return value, nil
	}
	return value, &responseFailure{statusCode: status, value: value}
}

func pageValues(data pageData) pageData {
	data.RequiredAddFields, data.OptionalAddFields = splitAddFieldViews(data.AddFields)
	return data
}

func editResponseFrom(data pageData, status int, redirectURL string) (editResponse, error) {
	response := editResponse{pageData: pageValues(data), statusCode: status, redirectURL: redirectURL}
	return responseResult(response, response.statusCode)
}

func addResponseFrom(data pageData, status int, redirectURL string) (addResponse, error) {
	response := addResponse{pageData: pageValues(data), statusCode: status, redirectURL: redirectURL}
	return responseResult(response, response.statusCode)
}

func moveResponseFrom(data pageData, status int, redirectURL string) (moveResponse, error) {
	response := moveResponse{pageData: pageValues(data), statusCode: status, redirectURL: redirectURL}
	return responseResult(response, response.statusCode)
}

func archiveResponseFrom(data pageData, status int, redirectURL string) (archiveResponse, error) {
	response := archiveResponse{pageData: pageValues(data), statusCode: status, redirectURL: redirectURL}
	return responseResult(response, response.statusCode)
}

func adaptorResponseEncoder(templates *template.Template) responseEncoder {
	return responseEncoder{templates: templates}
}

type responseEncoder struct {
	templates *template.Template
}

func (e responseEncoder) Encode(w http.ResponseWriter, r *http.Request, defaultStatus int, value any) error {
	switch response := value.(type) {
	case listResponse:
		return e.encodePage(w, r, response.pageData, response.statusCode, "Unable to render the page.", "")
	case itemResponse:
		if response.message != "" {
			return writeText(w, statusOrDefault(response.statusCode, defaultStatus), response.message)
		}
		return e.encodeTemplate(w, "entry-item", response, statusOrDefault(response.statusCode, defaultStatus), true, "Unable to render the item.")
	case editFormResponse:
		if response.message != "" {
			return writeText(w, statusOrDefault(response.statusCode, defaultStatus), response.message)
		}
		return e.encodeTemplate(w, response.templateName, response, statusOrDefault(response.statusCode, defaultStatus), true, "Unable to render the field.")
	case searchResponse:
		if response.message != "" {
			if response.retryAfter != "" {
				w.Header().Set("Retry-After", response.retryAfter)
			}
			return writeText(w, statusOrDefault(response.statusCode, defaultStatus), response.message)
		}
		return e.encodeTemplate(w, "search-results", response, statusOrDefault(response.statusCode, defaultStatus), true, "Unable to render search results.")
	case searchClearButtonResponse:
		if !response.Visible {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")
			w.WriteHeader(statusOrDefault(response.statusCode, defaultStatus))
			return nil
		}
		return e.encodeTemplate(w, "search-clear-button", response, statusOrDefault(response.statusCode, defaultStatus), false, "Unable to render the clear search button.")
	case liveReloadResponse:
		return writeEventStream(w, r, statusOrDefault(response.statusCode, defaultStatus))
	case editResponse:
		return e.encodePage(w, r, response.pageData, response.statusCode, "Unable to render the page.", response.redirectURL)
	case addResponse:
		return e.encodePage(w, r, response.pageData, response.statusCode, "Unable to render the page.", response.redirectURL)
	case moveResponse:
		return e.encodePage(w, r, response.pageData, response.statusCode, "Unable to render the page.", response.redirectURL)
	case archiveResponse:
		return e.encodePage(w, r, response.pageData, response.statusCode, "Unable to render the page.", response.redirectURL)
	case assetResponse:
		if response.message != "" {
			return writeText(w, statusOrDefault(response.statusCode, defaultStatus), response.message)
		}
		status := statusOrDefault(response.statusCode, defaultStatus)
		w.Header().Set("Content-Type", response.contentType)
		if status == http.StatusOK {
			w.Header().Set("Cache-Control", "public, max-age=3600")
		}
		w.WriteHeader(status)
		_, err := w.Write(response.body)
		return err
	default:
		return fmt.Errorf("unexpected response type %T", value)
	}
}

func (e responseEncoder) encodePage(w http.ResponseWriter, r *http.Request, data pageData, status int, errorMessage, redirectURL string) error {
	if redirectURL != "" {
		http.Redirect(w, r, redirectURL, statusOrDefault(status, http.StatusSeeOther))
		return nil
	}
	data = pageValues(data)
	templateName := "page"
	varyHTMX := false
	if isHTMX(r) {
		templateName = "workspace"
		varyHTMX = true
	}
	return e.encodeTemplate(w, templateName, data, statusOrDefault(status, http.StatusOK), varyHTMX, errorMessage)
}

func (e responseEncoder) encodeTemplate(w http.ResponseWriter, name string, data any, status int, varyHTMX bool, errorMessage string) error {
	var output strings.Builder
	if err := e.templates.ExecuteTemplate(&output, name, data); err != nil {
		return writeText(w, http.StatusInternalServerError, errorMessage)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if varyHTMX {
		w.Header().Set("Vary", "HX-Request")
	}
	w.WriteHeader(status)
	_, err := w.Write([]byte(output.String()))
	return err
}

func statusOrDefault(status, defaultStatus int) int {
	if status != 0 {
		return status
	}
	return defaultStatus
}

func writeText(w http.ResponseWriter, status int, message string) error {
	http.Error(w, message, status)
	return nil
}

func writeEventStream(w http.ResponseWriter, r *http.Request, status int) error {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE streaming is unavailable.", http.StatusInternalServerError)
		return nil
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(status)
	if _, err := fmt.Fprint(w, ": connected\n\n"); err != nil {
		return err
	}
	flusher.Flush()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return nil
		case <-ticker.C:
			if _, err := fmt.Fprint(w, ": keep-alive\n\n"); err != nil {
				return err
			}
			flusher.Flush()
		}
	}
}

type contextFlag uint8

const (
	htmxContextFlag contextFlag = iota + 1
	sameOriginContextFlag
)

func requestMetadata(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), htmxContextFlag, isHTMX(r))
		ctx = context.WithValue(ctx, sameOriginContextFlag, sameOrigin(r))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func isHTMXContext(ctx context.Context) bool {
	isHTMX, _ := ctx.Value(htmxContextFlag).(bool)
	return isHTMX
}

func isSameOriginContext(ctx context.Context) bool {
	sameOrigin, _ := ctx.Value(sameOriginContextFlag).(bool)
	return sameOrigin
}
