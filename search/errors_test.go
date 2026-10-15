package search

import (
	"errors"
	"io/fs"
	"testing"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

type localizableError interface {
	error
	LocalizeWith(*i18n.Localizer) (string, error)
}

func TestSearchErrors(t *testing.T) {
	t.Parallel()

	cause := fs.ErrPermission
	tests := []struct {
		name          string
		err           localizableError
		wantError     string
		wantLocalized string
		wantCause     error
	}{
		{
			name:          "nil source filesystem",
			err:           ErrSourceFilesystemIsNil,
			wantError:     "search: source filesystem is nil",
			wantLocalized: "search: source filesystem is nil",
		},
		{
			name:          "index is still building",
			err:           ErrSearchIndexIsBuilding,
			wantError:     "search: index is still being built",
			wantLocalized: "search: index is still being built",
		},
		{
			name:          "nil file change callback",
			err:           ErrFileChangeIsNil,
			wantError:     "search: file change is nil",
			wantLocalized: "search: file change is nil",
		},
		{
			name:          "zero value is unknown",
			err:           ErrUnknown,
			wantError:     "unknown search error code #0",
			wantLocalized: "unknown search error code #0",
		},
		{
			name:          "unknown error code",
			err:           Error(255),
			wantError:     "unknown search error code #255",
			wantLocalized: "unknown search error code #255",
		},
		{
			name:          "CUE parse failure",
			err:           ParseError{Path: "people.cue", Cause: cause},
			wantError:     `parse CUE file "people.cue" for search index: permission denied`,
			wantLocalized: "parse CUE file people.cue for search index: permission denied",
			wantCause:     cause,
		},
		{
			name:          "entry read failure",
			err:           ReadEntryForIndexError{Index: 2, Path: "people.cue", Err: cause},
			wantError:     `read entry 2 from "people.cue" for search index: permission denied`,
			wantLocalized: "read entry 2 from people.cue for search index: permission denied",
			wantCause:     cause,
		},
		{
			name:          "entry index failure",
			err:           IndexEntryError{Index: 2, Path: "people.cue", Err: cause},
			wantError:     `index entry 2 from "people.cue": permission denied`,
			wantLocalized: "index entry 2 from people.cue: permission denied",
			wantCause:     cause,
		},
		{
			name:          "source filesystem walk failure",
			err:           WalkSourceFilesystemError{Err: cause},
			wantError:     "walk source filesystem for search index: permission denied",
			wantLocalized: "walk source filesystem for search index: permission denied",
			wantCause:     cause,
		},
		{
			name:          "non-string entry key",
			err:           IndexEntryKeyError{KeyType: "int"},
			wantError:     "search index entry key has type int",
			wantLocalized: "search index entry key has type int",
		},
	}

	localizer := i18n.NewLocalizer(i18n.NewBundle(language.AmericanEnglish))
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.err.Error(); got != test.wantError {
				t.Errorf("Error() = %q, want %q", got, test.wantError)
			}
			localized, err := test.err.LocalizeWith(localizer)
			if err != nil {
				t.Fatalf("LocalizeWith() error = %v", err)
			}
			if localized != test.wantLocalized {
				t.Errorf("LocalizeWith() = %q, want %q", localized, test.wantLocalized)
			}
			if test.wantCause != nil && !errors.Is(test.err, test.wantCause) {
				t.Errorf("errors.Is(error, cause) = false, want true")
			}
		})
	}
}
