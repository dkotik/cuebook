package htmx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/dkotik/htadaptor"
)

func TestConfiguredAdaptorMiddlewareAppliesToRoutes(t *testing.T) {
	adaptor := htadaptor.New(htadaptor.WithMiddleware(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Adaptor-Middleware", "applied")
			next.ServeHTTP(w, r)
		})
	}))
	handler, err := New(fstest.MapFS{}, WithAdaptor(&adaptor))
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if got := response.Header().Get("X-Adaptor-Middleware"); got != "applied" {
		t.Errorf("adaptor middleware header = %q, want applied", got)
	}
}

func TestServeMuxPrefixAppliesToRoutesAndGeneratedURLs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		prefix     string
		requestURL string
		wantPrefix string
	}{
		{name: "root prefix", prefix: "/", requestURL: "http://example.test/"},
		{name: "nested prefix with trailing slash", prefix: "/catalog/", requestURL: "http://example.test/catalog/", wantPrefix: "/catalog"},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			handler, err := New(fstest.MapFS{
				"one.cue": &fstest.MapFile{Data: []byte(`[{Name: "One"}]`)},
			}, WithServeMuxPrefix(test.prefix))
			if err != nil {
				t.Fatal(err)
			}

			pageRequest := httptest.NewRequest(http.MethodGet, test.requestURL+"?file=one.cue", nil)
			pageResponse := httptest.NewRecorder()
			handler.ServeHTTP(pageResponse, pageRequest)
			if pageResponse.Code != http.StatusOK {
				t.Fatalf("page status = %d, want %d; body: %s", pageResponse.Code, http.StatusOK, pageResponse.Body.String())
			}

			for _, want := range []string{
				`data-route-prefix="` + test.wantPrefix + `"`,
				`href="` + test.wantPrefix + `/?file=one.cue"`,
				`href="` + test.wantPrefix + `/entry?`,
				`src="` + test.wantPrefix + `/assets/live-reload.js"`,
				`data-events-url="` + test.wantPrefix + `/events"`,
			} {
				if !strings.Contains(pageResponse.Body.String(), want) {
					t.Errorf("page does not contain %q; body: %s", want, pageResponse.Body.String())
				}
			}

			assetURL := test.wantPrefix + "/assets/app.css"
			assetRequest := httptest.NewRequest(http.MethodGet, "http://example.test"+assetURL, nil)
			assetResponse := httptest.NewRecorder()
			handler.ServeHTTP(assetResponse, assetRequest)
			if assetResponse.Code != http.StatusOK {
				t.Errorf("asset status = %d, want %d", assetResponse.Code, http.StatusOK)
			}
		})
	}
}
