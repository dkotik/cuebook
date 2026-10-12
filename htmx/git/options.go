package git

import (
	"errors"
	"fmt"
	"reflect"
)

type options struct {
	FileLinker FileLinker
}

// Option configures a Git changes handler.
type Option func(*options) error

// WithFileLinker supplies the function used to create a link for each changed
// file path.
func WithFileLinker(linker FileLinker) Option {
	return func(configured *options) error {
		if isNil(linker) {
			return errors.New("git: file linker is nil")
		}
		configured.FileLinker = linker
		return nil
	}
}

func resolveOptions(opts []Option) (options, error) {
	var configured options
	for index, option := range opts {
		if option == nil {
			return options{}, fmt.Errorf("git: option %d is nil", index+1)
		}
		if err := option(&configured); err != nil {
			return options{}, fmt.Errorf("git: failed to apply option %d: %w", index+1, err)
		}
	}
	if isNil(configured.FileLinker) {
		return options{}, errors.New("git: file linker is required")
	}
	return configured, nil
}

func isNil(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}
