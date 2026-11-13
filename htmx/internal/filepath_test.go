package internal

import (
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestFilepathExtractor(t *testing.T) {
	tests := []struct {
		name       string
		routeValue string
		pathValue  string
		suffix     string
		wantFile   string
		wantErr    bool
	}{
		{
			name:       "joins path segments and appends suffix",
			routeValue: "filepath",
			pathValue:  "collection/./notes/../book",
			suffix:     ".cue",
			wantFile:   "collection/book.cue",
		},
		{
			name:       "custom route value",
			routeValue: "asset",
			pathValue:  "collection//book",
			suffix:     ".md",
			wantFile:   "collection/book.md",
		},
		{
			name:       "empty suffix",
			routeValue: "filepath",
			pathValue:  "collection/book",
			wantFile:   "collection/book",
		},
		{
			name:       "empty route value",
			routeValue: "filepath",
			wantErr:    true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest("GET", "/", nil)
			request.SetPathValue(test.routeValue, test.pathValue)
			values := url.Values{}

			err := NewFilepathExtractor(test.routeValue, test.suffix).ExtractRequestValue(values, request)
			if (err != nil) != test.wantErr {
				t.Fatalf("ExtractRequestValue() error = %v, want error %t", err, test.wantErr)
			}
			if test.wantErr {
				if _, ok := values[test.routeValue]; ok {
					t.Errorf("route value set on error: %v", values[test.routeValue])
				}
				return
			}
			if got := values.Get(test.routeValue); got != test.wantFile {
				t.Errorf("route value = %q, want %q", got, test.wantFile)
			}
		})
	}
}
