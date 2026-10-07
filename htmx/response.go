package htmx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/dkotik/htadaptor"
)

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

func responseErrorHandler(encoder htadaptor.Encoder) htadaptor.ErrorHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		var failure *responseFailure
		if errors.As(err, &failure) {
			writeResponseFailure(w, r, failure, encoder)
			return
		}

		status := htadaptor.GetHyperTextStatusCode(err)
		var decodingError *htadaptor.DecodingError
		message := err.Error()
		if errors.As(err, &decodingError) {
			status = http.StatusBadRequest
			message = "The request is invalid."
		}
		http.Error(w, message, status)
	}
}

func writeResponseFailure(w http.ResponseWriter, r *http.Request, failure *responseFailure, encoder htadaptor.Encoder) {
	switch response := failure.value.(type) {
	case itemResponse:
		if response.message != "" {
			_ = writeText(w, failure.statusCode, response.message)
			return
		}
	case editFormResponse:
		if response.message != "" {
			_ = writeText(w, failure.statusCode, response.message)
			return
		}
	case searchResponse:
		if response.retryAfter != "" {
			w.Header().Set("Retry-After", response.retryAfter)
		}
		if response.message != "" {
			_ = writeText(w, failure.statusCode, response.message)
			return
		}
	case editResponse:
		if response.redirectURL != "" {
			http.Redirect(w, r, response.redirectURL, failure.statusCode)
			return
		}
	case addResponse:
		if response.redirectURL != "" {
			http.Redirect(w, r, response.redirectURL, failure.statusCode)
			return
		}
	case moveResponse:
		if response.redirectURL != "" {
			http.Redirect(w, r, response.redirectURL, failure.statusCode)
			return
		}
	case archiveResponse:
		if response.redirectURL != "" {
			http.Redirect(w, r, response.redirectURL, failure.statusCode)
			return
		}
	}
	if err := encoder.Encode(w, r, failure.statusCode, failure.value); err != nil {
		http.Error(w, "Unable to encode the response.", http.StatusInternalServerError)
	}
}

func writeText(w http.ResponseWriter, status int, message string) error {
	http.Error(w, message, status)
	return nil
}

func serveLiveReloadEvents(w http.ResponseWriter, r *http.Request) {
	_ = writeEventStream(w, r, http.StatusOK)
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

func isHTMXContext(ctx context.Context) bool {
	isHTMX, _ := ctx.Value(htmxContextFlag).(bool)
	return isHTMX
}

func isSameOriginContext(ctx context.Context) bool {
	sameOrigin, _ := ctx.Value(sameOriginContextFlag).(bool)
	return sameOrigin
}
