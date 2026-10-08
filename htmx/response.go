package htmx

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/dkotik/htadaptor"
)

type responseFailure struct {
	cause error
	value any
}

func (e *responseFailure) Error() string {
	return "domain response requires a non-success HTTP status"
}

func (e *responseFailure) Unwrap() error {
	return e.cause
}

type responseStatusError int

func (e responseStatusError) Error() string {
	return fmt.Sprintf("HTTP status %d", int(e))
}

func (e responseStatusError) HyperTextStatusCode() int {
	return int(e)
}

func responseErrorForStatus(status int) error {
	if status == 0 || status == http.StatusOK {
		return nil
	}
	return responseStatusError(status)
}

func responseResult[T any](value T, cause error) (T, error) {
	if cause == nil {
		return value, nil
	}
	return value, &responseFailure{cause: cause, value: value}
}

func pageValues(data pageData) pageData {
	data.RequiredAddFields, data.OptionalAddFields = splitAddFieldViews(data.AddFields)
	return data
}

func editResponseFrom(data pageData, status int) (editResponse, error) {
	response := editResponse{pageData: pageValues(data)}
	return responseResult(response, responseErrorForStatus(status))
}

func addResponseFrom(data pageData, status int) (addResponse, error) {
	response := addResponse{pageData: pageValues(data)}
	return responseResult(response, responseErrorForStatus(status))
}

func moveResponseFrom(data pageData, status int) (moveResponse, error) {
	response := moveResponse{pageData: pageValues(data)}
	return responseResult(response, responseErrorForStatus(status))
}

func archiveResponseFrom(data pageData, status int) (archiveResponse, error) {
	response := archiveResponse{pageData: pageValues(data)}
	return responseResult(response, responseErrorForStatus(status))
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
	status := htadaptor.GetHyperTextStatusCode(failure)
	switch response := failure.value.(type) {
	case itemResponse:
		if response.message != "" {
			_ = writeText(w, status, response.message)
			return
		}
	case editFormResponse:
		if response.message != "" {
			_ = writeText(w, status, response.message)
			return
		}
	case searchResponse:
		if response.retryAfter != "" {
			w.Header().Set("Retry-After", response.retryAfter)
		}
		if response.message != "" {
			_ = writeText(w, status, response.message)
			return
		}

	}
	if err := encoder.Encode(w, r, status, failure.value); err != nil {
		http.Error(w, "Unable to encode the response.", http.StatusInternalServerError)
	}
}

func writeText(w http.ResponseWriter, status int, message string) error {
	http.Error(w, message, status)
	return nil
}
