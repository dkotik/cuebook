package htmx

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/dkotik/htadaptor"
)

type httpStatusError struct {
	status  int
	message string
}

var _ htadaptor.Error = (*httpStatusError)(nil)

func (e *httpStatusError) Error() string {
	return e.message
}

func (e *httpStatusError) HyperTextStatusCode() int {
	return e.status
}

type retryAfterError struct {
	cause error
	value string
}

func (e *retryAfterError) Error() string {
	return e.cause.Error()
}

func (e *retryAfterError) Unwrap() error {
	return e.cause
}

func (e *retryAfterError) RetryAfter() string {
	return e.value
}

func responseErrorForStatus(status int, message string) error {
	if status == 0 || status == http.StatusOK {
		return nil
	}
	if message == "" {
		message = http.StatusText(status)
		if message == "" {
			message = fmt.Sprintf("HTTP status %d", status)
		}
	}
	if status == http.StatusInternalServerError {
		return errors.New(message)
	}
	return &httpStatusError{status: status, message: message}
}

func pageResponseMessage(data pageData) string {
	for _, message := range []string{data.Error, data.AddError, data.DocumentError} {
		if message != "" {
			return message
		}
	}
	return ""
}

func pageValues(data pageData) pageData {
	data.RequiredAddFields, data.OptionalAddFields = splitAddFieldViews(data.AddFields)
	return data
}

func editResponseFrom(data pageData, status int) (editResponse, error) {
	response := editResponse{pageData: pageValues(data)}
	return response, responseErrorForStatus(status, pageResponseMessage(data))
}

func addResponseFrom(data pageData, status int) (addResponse, error) {
	response := addResponse{pageData: pageValues(data)}
	return response, responseErrorForStatus(status, pageResponseMessage(data))
}

func moveResponseFrom(data pageData, status int) (moveResponse, error) {
	response := moveResponse{pageData: pageValues(data)}
	return response, responseErrorForStatus(status, pageResponseMessage(data))
}

func archiveResponseFrom(data pageData, status int) (archiveResponse, error) {
	response := archiveResponse{pageData: pageValues(data)}
	return response, responseErrorForStatus(status, pageResponseMessage(data))
}

func responseErrorHandler() htadaptor.ErrorHandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request, err error) {
		var decodingError *htadaptor.DecodingError
		message := err.Error()
		if errors.As(err, &decodingError) {
			message = "The request is invalid."
		}

		var retryAfter interface{ RetryAfter() string }
		if errors.As(err, &retryAfter) {
			if value := retryAfter.RetryAfter(); value != "" {
				w.Header().Set("Retry-After", value)
			}
		}

		http.Error(w, message, htadaptor.GetHyperTextStatusCode(err))
	}
}
