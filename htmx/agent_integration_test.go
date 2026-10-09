//go:build integration

package htmx_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/dkotik/cuebook/htmx"
	"github.com/dkotik/cuebook/htmx/agent"
)

type integrationGenerator struct{}

func (integrationGenerator) Generate(_ context.Context, _ []agent.Message, _ int) (string, error) {
	return "The local document says hello.", nil
}

func TestHTMXMountsAgentUnderPrefixedRoute(t *testing.T) {
	t.Parallel()

	assistant, err := agent.New(fstest.MapFS{
		"guide.md": {Data: []byte("The workspace guide says hello.")},
	}, agent.WithModel(integrationGenerator{}))
	if err != nil {
		t.Fatal(err)
	}
	defer assistant.Close(context.Background())

	handler, err := htmx.New(fstest.MapFS{}, htmx.WithAgent(assistant), htmx.WithServeMuxPrefix("/catalog"))
	if err != nil {
		t.Fatal(err)
	}

	page := perform(handler, http.MethodGet, "http://example.test/catalog/", nil, nil)
	if page.Code != http.StatusOK {
		t.Fatalf("page status = %d, want %d; body: %s", page.Code, http.StatusOK, page.Body.String())
	}
	for _, expected := range []string{
		`href="/catalog/agent/assets/agent.css"`,
		`src="/catalog/agent/assets/agent.js"`,
		`hx-get="/catalog/agent/widget"`,
	} {
		if !strings.Contains(page.Body.String(), expected) {
			t.Errorf("page missing %q; body: %s", expected, page.Body.String())
		}
	}

	widget := perform(handler, http.MethodGet, "http://example.test/catalog/agent/widget", nil, nil)
	if widget.Code != http.StatusOK || !strings.Contains(widget.Body.String(), `hx-post="/catalog/agent/chat"`) {
		t.Fatalf("mounted widget = %d %s", widget.Code, widget.Body.String())
	}

	asset := perform(handler, http.MethodGet, "http://example.test/catalog/agent/assets/agent.css", nil, nil)
	if asset.Code != http.StatusOK || !strings.Contains(asset.Header().Get("Content-Type"), "text/css") {
		t.Errorf("mounted asset = %d %q", asset.Code, asset.Header().Get("Content-Type"))
	}

	chat := perform(handler, http.MethodPost, "http://example.test/catalog/agent/chat", strings.NewReader(url.Values{"message": {"hello"}}.Encode()), func(request *http.Request) {
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request.Header.Set("Origin", "http://example.test")
	})
	if chat.Code != http.StatusOK || !strings.Contains(chat.Body.String(), "The local document says hello.") {
		t.Errorf("mounted chat = %d %s", chat.Code, chat.Body.String())
	}
}

func TestHTMXDoesNotRenderAgentWhenDisabled(t *testing.T) {
	t.Parallel()

	handler, err := htmx.New(fstest.MapFS{})
	if err != nil {
		t.Fatal(err)
	}
	page := perform(handler, http.MethodGet, "http://example.test/", nil, nil)
	if page.Code != http.StatusOK {
		t.Fatalf("page status = %d, want %d; body: %s", page.Code, http.StatusOK, page.Body.String())
	}
	for _, absent := range []string{"agent-widget-loader", "/agent/assets/agent.css", "/agent/assets/agent.js"} {
		if strings.Contains(page.Body.String(), absent) {
			t.Errorf("disabled page contains %q", absent)
		}
	}
}

func perform(handler http.Handler, method, target string, body io.Reader, configure func(*http.Request)) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, target, body)
	if configure != nil {
		configure(request)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
