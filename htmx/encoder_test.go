package htmx

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dkotik/htadaptor"
)

type flashResponse struct{ message string }

func (response flashResponse) GetFlashMessage() string { return response.message }

type redirectResponse struct{ location string }

func (response redirectResponse) GetRedirect() string { return response.location }

type flashRedirectResponse struct {
	message  string
	location string
}

func (response flashRedirectResponse) GetFlashMessage() string { return response.message }
func (response flashRedirectResponse) GetRedirect() string     { return response.location }

func TestNewEncoder(t *testing.T) {
	t.Parallel()

	encodeErr := errors.New("encode failed")
	tests := []struct {
		name       string
		response   any
		wantFlash  string
		wantTarget string
	}{
		{
			name:     "plain response",
			response: struct{}{},
		},
		{
			name:      "flash response",
			response:  flashResponse{message: "Entry archived."},
			wantFlash: "Entry archived.",
		},
		{
			name:       "redirect response",
			response:   redirectResponse{location: "/entries"},
			wantTarget: "/entries",
		},
		{
			name:       "flash and redirect response",
			response:   flashRedirectResponse{message: "Saved.", location: "/entries"},
			wantFlash:  "Saved.",
			wantTarget: "/entries",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			called := false
			wrap := htadaptor.EncoderFunc(func(w http.ResponseWriter, r *http.Request, status int, response any) error {
				called = true
				if got := w.Header().Get("HX-Trigger"); got != test.wantFlash {
					t.Errorf("HX-Trigger = %q, want %q", got, test.wantFlash)
				}
				if got := w.Header().Get("HX-Redirect"); got != test.wantTarget {
					t.Errorf("HX-Redirect = %q, want %q", got, test.wantTarget)
				}
				if status != http.StatusAccepted {
					t.Errorf("status = %d, want %d", status, http.StatusAccepted)
				}
				if response != test.response {
					t.Errorf("response = %#v, want %#v", response, test.response)
				}
				return encodeErr
			})

			gotErr := NewEncoder(wrap).Encode(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil), http.StatusAccepted, test.response)
			if !errors.Is(gotErr, encodeErr) {
				t.Errorf("Encode() error = %v, want %v", gotErr, encodeErr)
			}
			if !called {
				t.Error("wrapped encoder was not called")
			}
		})
	}
}
