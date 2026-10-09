package agent

import (
	"strings"
	"testing"
	"time"
)

func TestResolveOptionsValidatesModelAndBounds(t *testing.T) {
	t.Parallel()

	generator := &fakeGenerator{response: "answer"}
	tests := []struct {
		name    string
		options []Option
		wantErr string
	}{
		{name: "missing model", wantErr: "configure a model"},
		{name: "valid injected model", options: []Option{WithModel(generator)}},
		{name: "default Kronk model", options: []Option{WithDefaultModel()}},
		{name: "nil option", options: []Option{nil}, wantErr: "option 1 is nil"},
		{name: "duplicate model", options: []Option{WithModel(generator), WithDefaultModel()}, wantErr: "model is already configured"},
		{name: "invalid prefix", options: []Option{WithModel(generator), WithServeMuxPrefix("/../bad")}, wantErr: "invalid serve mux prefix"},
		{name: "invalid context limits", options: []Option{WithModel(generator), WithContextLimits(100, 80, 32, 2)}, wantErr: "must not exceed"},
		{name: "invalid output tokens", options: []Option{WithModel(generator), WithMaxOutputTokens(0)}, wantErr: "output token limit"},
		{name: "invalid session limits", options: []Option{WithModel(generator), WithSessionLimits(1, 1, time.Minute)}, wantErr: "at least one turn"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := resolveOptions(test.options)
			if test.wantErr == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("error = %v, want message containing %q", err, test.wantErr)
			}
		})
	}
}

func TestWithModelRejectsNil(t *testing.T) {
	t.Parallel()

	_, err := resolveOptions([]Option{WithModel(nil)})
	if err == nil || !strings.Contains(err.Error(), "model is nil") {
		t.Fatalf("error = %v, want nil-model error", err)
	}
}

func TestNormalizePrefix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "empty", input: "", want: ""},
		{name: "root", input: "/", want: ""},
		{name: "leading slash", input: "/assistant", want: "/assistant"},
		{name: "no leading slash or trailing slash", input: "assistant/", want: "/assistant"},
		{name: "invalid traversal", input: "/../assistant", wantErr: true},
		{name: "invalid query", input: "/assistant?next", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := normalizePrefix(test.input)
			if test.wantErr {
				if err == nil {
					t.Fatal("expected a prefix error")
				}
				return
			}
			if err != nil || got != test.want {
				t.Errorf("normalizePrefix(%q) = (%q, %v), want %q", test.input, got, err, test.want)
			}
		})
	}
}
