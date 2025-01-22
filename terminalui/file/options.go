package file

import (
	"errors"
	"slices"
)

type options struct {
	AllowedExtensions []string
	OnSelect          Command
}

type Option func(*options) error

func WithAllowedFileExtensions(exts ...string) Option {
	return func(o *options) error {
		if len(exts) == 0 {
			return errors.New("no file extensions provided")
		}
		more := make([]string, 0, len(exts))
		for _, ext := range exts {
			if ext == "" {
				return errors.New("empty file extension provided")
			}
			if ext[0] != '.' {
				return errors.New("a file extension must begin with a dot")
			}
			if slices.Index(o.AllowedExtensions, ext) != -1 {
				continue
			}
			more = append(more, ext)
		}
		o.AllowedExtensions = append(o.AllowedExtensions, more...)
		return nil
	}
}

func WithOnSelect(cmd Command) Option {
	return func(o *options) error {
		if cmd == nil {
			return errors.New("nil selection command")
		}
		if o.OnSelect != nil {
			return errors.New("selection command already set")
		}
		o.OnSelect = cmd
		return nil
	}
}
