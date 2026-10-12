package git

import (
	"errors"
	"strings"
	"testing"
)

type testFileLinker struct {
	prefix string
}

func (linker testFileLinker) GetFileLink(path string) (string, error) {
	return linker.prefix + path, nil
}

func TestResolveOptions(t *testing.T) {
	t.Parallel()

	var nilOption Option
	var nilLinker *testFileLinker
	tests := []struct {
		name     string
		options  []Option
		wantErr  string
		wantLink string
	}{
		{
			name:    "file linker is required",
			wantErr: "file linker is required",
		},
		{
			name:    "nil option is rejected",
			options: []Option{nilOption},
			wantErr: "option 1 is nil",
		},
		{
			name:    "nil linker is rejected",
			options: []Option{WithFileLinker(nil)},
			wantErr: "file linker is nil",
		},
		{
			name:    "typed nil linker is rejected",
			options: []Option{WithFileLinker(nilLinker)},
			wantErr: "file linker is nil",
		},
		{
			name:     "valid linker is accepted",
			options:  []Option{WithFileLinker(testFileLinker{prefix: "/files/"})},
			wantLink: "/files/example.txt",
		},
		{
			name:     "last linker option wins",
			options:  []Option{WithFileLinker(testFileLinker{prefix: "/first/"}), WithFileLinker(testFileLinker{prefix: "/last/"})},
			wantLink: "/last/example.txt",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			configured, err := resolveOptions(test.options)
			if test.wantErr == "" {
				if err != nil {
					t.Fatalf("resolveOptions() error = %v", err)
				}
				if test.wantLink != "" {
					got, err := configured.FileLinker.GetFileLink("example.txt")
					if err != nil || got != test.wantLink {
						t.Fatalf("resolved linker result = %q, error = %v; want %q", got, err, test.wantLink)
					}
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("resolveOptions() error = %v, want it to contain %q", err, test.wantErr)
			}
		})
	}
}

func TestWithFileLinkerOptionErrorIsWrapped(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("invalid linker")
	option := Option(func(*options) error { return wantErr })
	_, err := resolveOptions([]Option{option})
	if !errors.Is(err, wantErr) {
		t.Fatalf("resolveOptions() error = %v, want wrapped %v", err, wantErr)
	}
}
