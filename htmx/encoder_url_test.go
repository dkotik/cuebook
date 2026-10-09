package htmx

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dkotik/htadaptor"
)

type replacementResponse struct {
	location  string
	flashOnly bool
}

func (response replacementResponse) GetURLReplacement() string { return response.location }
func (response replacementResponse) GetFlashMessage() (string, bool) {
	if response.flashOnly {
		return "Invalid field.", false
	}
	return "", true
}

func TestEncoderReplacesURLWithoutHTMXRedirect(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		htmx            bool
		response        replacementResponse
		wantStatus      int
		wantReplacement string
		wantLocation    string
		wantWrapped     bool
	}{
		{name: "HTMX updates history and encodes fragment", htmx: true, response: replacementResponse{location: "/entry?head=1&path=people.cue&tail=30"}, wantStatus: http.StatusAccepted, wantReplacement: "/entry?head=1&path=people.cue&tail=30", wantWrapped: true},
		{name: "browser uses POST redirect GET to same entry", response: replacementResponse{location: "/entry?head=1&path=people.cue&tail=30"}, wantStatus: http.StatusSeeOther, wantLocation: "/entry?head=1&path=people.cue&tail=30"},
		{name: "empty location encodes normally", htmx: true, wantStatus: http.StatusAccepted, wantWrapped: true},
		{name: "flash only stops before replacing URL", htmx: true, response: replacementResponse{location: "/entry", flashOnly: true}, wantStatus: http.StatusNoContent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			called := false
			encodeErr := errors.New("template error")
			wrap := htadaptor.EncoderFunc(func(w http.ResponseWriter, r *http.Request, status int, response any) error {
				called = true
				w.WriteHeader(status)
				return encodeErr
			})
			request := httptest.NewRequest(http.MethodPost, "/edit", nil)
			if tt.htmx {
				request.Header.Set("HX-Request", "true")
			}
			response := httptest.NewRecorder()
			err := NewEncoder(wrap).Encode(response, request, http.StatusAccepted, tt.response)
			if called != tt.wantWrapped {
				t.Errorf("wrapped encoder called = %t, want %t", called, tt.wantWrapped)
			}
			if tt.wantWrapped && !errors.Is(err, encodeErr) {
				t.Errorf("encode error = %v, want %v", err, encodeErr)
			}
			if !tt.wantWrapped && err != nil {
				t.Errorf("unexpected encode error: %v", err)
			}
			if response.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", response.Code, tt.wantStatus)
			}
			for header, want := range map[string]string{"HX-Replace-Url": tt.wantReplacement, "Location": tt.wantLocation, "HX-Redirect": ""} {
				if got := response.Header().Get(header); got != want {
					t.Errorf("%s = %q, want %q", header, got, want)
				}
			}
		})
	}
}
