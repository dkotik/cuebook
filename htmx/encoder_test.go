package htmx

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dkotik/htadaptor"
)

type flashResponse struct {
	message          string
	continueEncoding bool
}

func (response flashResponse) GetFlashMessage() (string, bool) {
	return response.message, response.continueEncoding
}

type urlReplacementResponse struct{ location string }

func (response urlReplacementResponse) GetURLReplacement() string { return response.location }

type flashURLReplacementResponse struct {
	message          string
	location         string
	continueEncoding bool
}

func (response flashURLReplacementResponse) GetFlashMessage() (string, bool) {
	return response.message, response.continueEncoding
}

func (response flashURLReplacementResponse) GetURLReplacement() string { return response.location }

func TestNewEncoder(t *testing.T) {
	t.Parallel()

	encodeErr := errors.New("encode failed")
	tests := []struct {
		name          string
		response      any
		wantFlash     string
		wantTarget    string
		wantWrapped   bool
		wantEncodeErr bool
	}{
		{
			name:          "plain response",
			response:      struct{}{},
			wantWrapped:   true,
			wantEncodeErr: true,
		},
		{
			name:          "flash response continues encoding",
			response:      flashResponse{message: "Entry archived.", continueEncoding: true},
			wantFlash:     "Entry archived.",
			wantWrapped:   true,
			wantEncodeErr: true,
		},
		{
			name:          "URL replacement response",
			response:      urlReplacementResponse{location: "/entries"},
			wantTarget:    "/entries",
			wantWrapped:   true,
			wantEncodeErr: true,
		},
		{
			name:          "flash and URL replacement response continues encoding",
			response:      flashURLReplacementResponse{message: "Saved.", location: "/entries", continueEncoding: true},
			wantFlash:     "Saved.",
			wantTarget:    "/entries",
			wantWrapped:   true,
			wantEncodeErr: true,
		},
		{
			name:        "flash-only response stops encoding",
			response:    flashResponse{message: "Entry archived."},
			wantFlash:   "Entry archived.",
			wantWrapped: false,
		},
		{
			name:        "flash safely encodes punctuation unicode and newlines",
			response:    flashResponse{message: "Invalid, \"CUE\"\nfield: café <script>"},
			wantFlash:   "Invalid, \"CUE\"\nfield: café <script>",
			wantWrapped: false,
		},
		{
			name:        "flash and URL replacement response stops before replacing URL",
			response:    flashURLReplacementResponse{message: "Saved.", location: "/entries"},
			wantFlash:   "Saved.",
			wantWrapped: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			called := false
			wrap := htadaptor.EncoderFunc(func(w http.ResponseWriter, r *http.Request, status int, response any) error {
				called = true

				if got := w.Header().Get("HX-Replace-Url"); got != test.wantTarget {
					t.Errorf("HX-Replace-Url = %q, want %q", got, test.wantTarget)
				}
				if status != http.StatusAccepted {
					t.Errorf("status = %d, want %d", status, http.StatusAccepted)
				}
				if response != test.response {
					t.Errorf("response = %#v, want %#v", response, test.response)
				}
				return encodeErr
			})

			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request.Header.Set("HX-Request", "true")
			gotErr := NewEncoder(wrap).Encode(response, request, http.StatusAccepted, test.response)
			if test.wantEncodeErr && !errors.Is(gotErr, encodeErr) {
				t.Errorf("Encode() error = %v, want %v", gotErr, encodeErr)
			}
			if !test.wantEncodeErr && gotErr != nil {
				t.Errorf("Encode() error = %v, want nil", gotErr)
			}
			if called != test.wantWrapped {
				t.Errorf("wrapped encoder called = %t, want %t", called, test.wantWrapped)
			}
			if test.wantFlash != "" {
				var trigger map[string]struct {
					Message string `json:"message"`
				}
				if err := json.Unmarshal([]byte(response.Header().Get("HX-Trigger")), &trigger); err != nil {
					t.Fatalf("invalid flash trigger: %v", err)
				}
				if got := trigger["cuebook:flash"].Message; got != test.wantFlash {
					t.Errorf("flash = %q, want %q", got, test.wantFlash)
				}
			} else if got := response.Header().Get("HX-Trigger"); got != "" {
				t.Errorf("unexpected flash trigger: %q", got)
			}
			if !test.wantWrapped && response.Code != http.StatusNoContent {
				t.Errorf("response status = %d, want %d", response.Code, http.StatusNoContent)
			}
			if !test.wantWrapped && response.Body.Len() != 0 {
				t.Errorf("flash-only response has a body: %s", response.Body.String())
			}
			if got := response.Header().Get("HX-Replace-Url"); got != test.wantTarget {
				t.Errorf("HX-Replace-Url = %q, want %q", got, test.wantTarget)
			}
			for _, header := range []string{"HX-Redirect", "HX-Location", "Location"} {
				if got := response.Header().Get(header); got != "" {
					t.Errorf("unexpected %s = %q", header, got)
				}
			}
		})
	}
}
