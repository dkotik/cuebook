package htmx

import (
	"errors"
	"net/http"

	"github.com/dkotik/htadaptor"
)

func routeAdaptorOptions(errorEncoder htadaptor.Encoder, routeOptions ...htadaptor.Option) []htadaptor.Option {
	options := []htadaptor.Option{
		htadaptor.WithErrorHandler(htadaptor.ErrorHandlerFunc(func(w http.ResponseWriter, r *http.Request, err error) {
			var responseFailure *responseFailure
			if errors.As(err, &responseFailure) {
				if encodeErr := errorEncoder.Encode(w, r, responseFailure.statusCode, responseFailure.value); encodeErr != nil {
					http.Error(w, "Unable to encode the response.", http.StatusInternalServerError)
				}
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
		})),
		htadaptor.WithMiddleware(requestMetadata),
		htadaptor.WithReadLimit(maxRequestBodySize),
	}
	return append(options, routeOptions...)
}

func formRouteAdaptorOptions(errorEncoder htadaptor.Encoder, routeOptions ...htadaptor.Option) []htadaptor.Option {
	options := []htadaptor.Option{htadaptor.WithUnsafeDecoder(requestDecoder{})}
	return routeAdaptorOptions(errorEncoder, append(options, routeOptions...)...)
}

func templateResponseHeaders(varyHTMX bool) htadaptor.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store")
			if varyHTMX {
				w.Header().Set("Vary", "HX-Request")
			}
			next.ServeHTTP(w, r)
		})
	}
}
