package agent

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"
)

type fakeGenerator struct {
	mu       sync.Mutex
	response string
	err      error
	calls    [][]Message
}

type blockingGenerator struct {
	started chan struct{}
}

func (generator *blockingGenerator) Generate(ctx context.Context, _ []Message, _ int) (string, error) {
	close(generator.started)
	<-ctx.Done()
	return "", ctx.Err()
}

func (generator *fakeGenerator) Generate(_ context.Context, messages []Message, _ int) (string, error) {
	generator.mu.Lock()
	defer generator.mu.Unlock()
	generator.calls = append(generator.calls, transcript(messages))
	return generator.response, generator.err
}

func (generator *fakeGenerator) recordedCalls() [][]Message {
	generator.mu.Lock()
	defer generator.mu.Unlock()
	calls := make([][]Message, len(generator.calls))
	for index, call := range generator.calls {
		calls[index] = transcript(call)
	}
	return calls
}

func TestAgentWidgetAndChatUseEmbeddedAssetsAndContext(t *testing.T) {
	t.Parallel()

	generator := &fakeGenerator{response: "<script>alert('unsafe')</script>"}
	handler, err := New(fstest.MapFS{
		"docs/alpha.md": {Data: []byte("Alpha is the project codename.")},
		"ignored.bin":   {Data: []byte("never send this")},
	}, WithModel(generator), WithServeMuxPrefix("/catalog/agent/"))
	if err != nil {
		t.Fatal(err)
	}
	defer handler.Close(context.Background())

	widgetResponse := serveAgentRequest(handler, http.MethodGet, "http://example.test/catalog/agent/widget", nil, nil)
	if widgetResponse.Code != http.StatusOK {
		t.Fatalf("widget status = %d, want %d; body: %s", widgetResponse.Code, http.StatusOK, widgetResponse.Body.String())
	}
	for _, expected := range []string{
		`<cuebook-agent`,
		`action="/catalog/agent/chat"`,
		`hx-post="/catalog/agent/chat"`,
		`data-agent-dock="left"`,
		`role="log"`,
		`aria-label="Conversation"`,
		`for="agent-message"`,
		`aria-label="Open Cuebook assistant"`,
	} {
		if !strings.Contains(widgetResponse.Body.String(), expected) {
			t.Errorf("widget does not contain %q; body: %s", expected, widgetResponse.Body.String())
		}
	}

	assetResponse := serveAgentRequest(handler, http.MethodGet, "http://example.test/catalog/agent/assets/agent.js", nil, nil)
	if assetResponse.Code != http.StatusOK || !strings.Contains(assetResponse.Header().Get("Content-Type"), "javascript") {
		t.Errorf("asset response = %d %q", assetResponse.Code, assetResponse.Header().Get("Content-Type"))
	}
	for _, expected := range []string{"startDrag(event)", "localStorage.setItem", "ResizeObserver", "this.dock(button.dataset.agentDock)"} {
		if !strings.Contains(assetResponse.Body.String(), expected) {
			t.Errorf("embedded JavaScript does not contain %q", expected)
		}
	}

	styleResponse := serveAgentRequest(handler, http.MethodGet, "http://example.test/catalog/agent/assets/agent.css", nil, nil)
	if styleResponse.Code != http.StatusOK || !strings.Contains(styleResponse.Header().Get("Content-Type"), "text/css") {
		t.Errorf("style response = %d %q", styleResponse.Code, styleResponse.Header().Get("Content-Type"))
	}
	for _, expected := range []string{"resize: both", "@media (max-width: 40rem)", "prefers-reduced-motion: no-preference", ":focus-visible"} {
		if !strings.Contains(styleResponse.Body.String(), expected) {
			t.Errorf("embedded CSS does not contain %q", expected)
		}
	}

	form := url.Values{"message": {"Explain alpha"}}
	chatResponse := postAgentMessage(handler, "http://example.test/catalog/agent/chat", form, nil, "http://example.test")
	if chatResponse.Code != http.StatusOK {
		t.Fatalf("chat status = %d, want %d; body: %s", chatResponse.Code, http.StatusOK, chatResponse.Body.String())
	}
	if !strings.Contains(chatResponse.Body.String(), "docs/alpha.md") {
		t.Errorf("transcript missing context citation: %s", chatResponse.Body.String())
	}
	if strings.Contains(chatResponse.Body.String(), "<script>alert") || !strings.Contains(chatResponse.Body.String(), "&lt;script&gt;") {
		t.Errorf("model response was not safely escaped: %s", chatResponse.Body.String())
	}
	cookie := chatResponse.Result().Cookies()
	if len(cookie) != 1 || cookie[0].Name != sessionCookieName || !cookie[0].HttpOnly || cookie[0].Path != "/catalog/agent" {
		t.Errorf("session cookie = %#v", cookie)
	}

	calls := generator.recordedCalls()
	if len(calls) != 1 || len(calls[0]) != 2 {
		t.Fatalf("generator calls = %#v", calls)
	}
	if calls[0][0].Role != "system" || !strings.Contains(calls[0][0].Content, "docs/alpha.md") || strings.Contains(calls[0][0].Content, "never send this") {
		t.Errorf("incorrect generation context: %#v", calls[0][0])
	}
	if calls[0][1].Role != "user" || calls[0][1].Content != "Explain alpha" {
		t.Errorf("user prompt = %#v", calls[0][1])
	}
}

func TestChatMaintainsIsolatedSessionHistory(t *testing.T) {
	t.Parallel()

	generator := &fakeGenerator{response: "answer"}
	handler, err := New(fstest.MapFS{"note.txt": {Data: []byte("workspace notes")}}, WithModel(generator))
	if err != nil {
		t.Fatal(err)
	}
	defer handler.Close(context.Background())

	first := postAgentMessage(handler, "http://example.test/chat", url.Values{"message": {"first question"}}, nil, "http://example.test")
	if first.Code != http.StatusOK {
		t.Fatalf("first status = %d; body: %s", first.Code, first.Body.String())
	}
	cookies := first.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("first response did not create a session cookie")
	}
	second := postAgentMessage(handler, "http://example.test/chat", url.Values{"message": {"follow-up"}}, cookies[0], "http://example.test")
	if second.Code != http.StatusOK {
		t.Fatalf("second status = %d; body: %s", second.Code, second.Body.String())
	}
	isolated := postAgentMessage(handler, "http://example.test/chat", url.Values{"message": {"separate"}}, nil, "http://example.test")
	if isolated.Code != http.StatusOK {
		t.Fatalf("isolated status = %d; body: %s", isolated.Code, isolated.Body.String())
	}

	calls := generator.recordedCalls()
	if len(calls) != 3 {
		t.Fatalf("recorded %d model calls, want 3", len(calls))
	}
	if len(calls[1]) != 4 || calls[1][1].Content != "first question" || calls[1][2].Role != "assistant" || calls[1][3].Content != "follow-up" {
		t.Errorf("follow-up did not receive the previous turn: %#v", calls[1])
	}
	if len(calls[2]) != 2 || calls[2][1].Content != "separate" {
		t.Errorf("new session received another session's history: %#v", calls[2])
	}
}

func TestChatRejectsInvalidRequests(t *testing.T) {
	t.Parallel()

	generator := &fakeGenerator{response: "answer"}
	handler, err := New(fstest.MapFS{}, WithModel(generator), WithMaxMessageBytes(8))
	if err != nil {
		t.Fatal(err)
	}
	defer handler.Close(context.Background())

	tests := []struct {
		name        string
		body        string
		contentType string
		origin      string
		wantStatus  int
	}{
		{name: "empty message", body: "message=%20", contentType: "application/x-www-form-urlencoded", origin: "http://example.test", wantStatus: http.StatusBadRequest},
		{name: "oversized message", body: "message=123456789", contentType: "application/x-www-form-urlencoded", origin: "http://example.test", wantStatus: http.StatusRequestEntityTooLarge},
		{name: "cross origin", body: "message=question", contentType: "application/x-www-form-urlencoded", origin: "https://attacker.test", wantStatus: http.StatusForbidden},
		{name: "wrong content type", body: `{ "message": "question" }`, contentType: "application/json", origin: "http://example.test", wantStatus: http.StatusUnsupportedMediaType},
		{name: "missing origin with cross-site metadata", body: "message=question", contentType: "application/x-www-form-urlencoded", wantStatus: http.StatusForbidden},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "http://example.test/chat", strings.NewReader(test.body))
			if test.contentType != "" {
				request.Header.Set("Content-Type", test.contentType)
			}
			if test.origin != "" {
				request.Header.Set("Origin", test.origin)
			} else {
				request.Header.Set("Sec-Fetch-Site", "cross-site")
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Errorf("status = %d, want %d; body: %s", response.Code, test.wantStatus, response.Body.String())
			}
		})
	}
	if calls := generator.recordedCalls(); len(calls) != 0 {
		t.Errorf("invalid requests reached model: %#v", calls)
	}
}

func TestChatPassesRequestCancellationToGenerator(t *testing.T) {
	generator := &blockingGenerator{started: make(chan struct{})}
	handler, err := New(fstest.MapFS{}, WithModel(generator), WithRequestTimeout(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	defer handler.Close(context.Background())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	request := httptest.NewRequest(http.MethodPost, "http://example.test/chat", strings.NewReader("message=question")).WithContext(ctx)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Origin", "http://example.test")
	response := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		handler.ServeHTTP(response, request)
		close(done)
	}()

	select {
	case <-generator.started:
	case <-time.After(time.Second):
		t.Fatal("generator was not called")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("handler did not return after request cancellation")
	}
	if response.Code != http.StatusGatewayTimeout {
		t.Errorf("status = %d, want %d; body: %s", response.Code, http.StatusGatewayTimeout, response.Body.String())
	}
}

func TestChatHidesModelErrors(t *testing.T) {
	t.Parallel()

	generator := &fakeGenerator{err: errors.New("private native runtime detail")}
	handler, err := New(fstest.MapFS{}, WithModel(generator))
	if err != nil {
		t.Fatal(err)
	}
	defer handler.Close(context.Background())

	response := postAgentMessage(handler, "http://example.test/chat", url.Values{"message": {"question"}}, nil, "http://example.test")
	if response.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusBadGateway, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "could not generate") || strings.Contains(response.Body.String(), "private native runtime detail") {
		t.Errorf("response leaks model error or omits safe explanation: %s", response.Body.String())
	}
}

func TestAgentMountOverridesConfiguredURLPrefix(t *testing.T) {
	t.Parallel()

	handler, err := New(fstest.MapFS{}, WithModel(&fakeGenerator{response: "ok"}), WithServeMuxPrefix("/standalone"))
	if err != nil {
		t.Fatal(err)
	}
	defer handler.Close(context.Background())

	mounted := http.StripPrefix("/app/agent", handler.Mount("/app/agent"))
	response := serveAgentRequest(mounted, http.MethodGet, "http://example.test/app/agent/widget", nil, nil)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `action="/app/agent/chat"`) {
		t.Fatalf("mounted widget = %d %s", response.Code, response.Body.String())
	}
}

func TestAppendTurnKeepsNewestCompleteTurns(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		history    []Message
		max        int
		wantLength int
		wantFirst  string
	}{
		{name: "within limit", history: []Message{{Role: "user", Content: "old"}, {Role: "assistant", Content: "reply"}}, max: 6, wantLength: 4, wantFirst: "old"},
		{name: "drops oldest complete turn", history: []Message{{Role: "user", Content: "old"}, {Role: "assistant", Content: "old reply"}}, max: 2, wantLength: 2, wantFirst: "new"},
		{name: "odd limit keeps complete pair", max: 3, wantLength: 2, wantFirst: "new"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := appendTurn(test.history, Message{Role: "user", Content: "new"}, Message{Role: "assistant", Content: "new reply"}, test.max)
			if len(got) != test.wantLength || got[0].Content != test.wantFirst {
				t.Errorf("appendTurn = %#v, want len %d and first content %q", got, test.wantLength, test.wantFirst)
			}
		})
	}
}

func TestSessionStoreExpiresAndCapsSessions(t *testing.T) {
	t.Parallel()

	store := newSessionStore(1, time.Minute)
	firstID, _, firstCreated, err := store.get("", time.Unix(100, 0))
	if err != nil || !firstCreated {
		t.Fatalf("first session = %q, created %v, err %v", firstID, firstCreated, err)
	}
	secondID, _, secondCreated, err := store.get("", time.Unix(110, 0))
	if err != nil || !secondCreated || firstID == secondID {
		t.Fatalf("second session = %q, created %v, err %v", secondID, secondCreated, err)
	}
	if got := store.count(); got != 1 {
		t.Fatalf("session count = %d, want 1", got)
	}
	_, _, createdAfterExpiry, err := store.get(secondID, time.Unix(200, 0))
	if err != nil || !createdAfterExpiry {
		t.Errorf("expired session created = %v, err %v; want a new session", createdAfterExpiry, err)
	}
}

func serveAgentRequest(handler http.Handler, method, target string, body io.Reader, cookie *http.Cookie) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, target, body)
	if cookie != nil {
		request.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func postAgentMessage(handler http.Handler, target string, values url.Values, cookie *http.Cookie, origin string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, target, strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if origin != "" {
		request.Header.Set("Origin", origin)
	}
	request.Header.Set("HX-Request", "true")
	if cookie != nil {
		request.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
