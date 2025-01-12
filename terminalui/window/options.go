package window

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

type options struct {
	commandContext context.Context
	stack          []tea.Model
	watchers       []tea.Model
	lcBundle       *i18n.Bundle
	logger         *slog.Logger
}

type Option func(*options) error

func WithCommandContext(ctx context.Context) Option {
	return func(o *options) error {
		if ctx == nil {
			return errors.New("command context is nil")
		}
		if o.commandContext != nil {
			return errors.New("command context is already set")
		}
		o.commandContext = ctx
		return nil
	}
}

func WithInitialModels(models ...tea.Model) Option {
	return func(o *options) error {
		for i, model := range models {
			if model == nil {
				return fmt.Errorf("initial model #%d is nil", i)
			}
		}
		o.stack = append(o.stack, models...)
		return nil
	}
}

// WithWatchers adds invisible [tea.Model]s that issue commands but never render output.
func WithWatchers(models ...tea.Model) Option {
	return func(o *options) error {
		for i, model := range models {
			if model == nil {
				return fmt.Errorf("watcher model #%d is nil", i)
			}
		}
		o.watchers = append(o.watchers, models...)
		return nil
	}
}

func WithLocalization(b *i18n.Bundle) Option {
	return func(o *options) error {
		if b == nil {
			return errors.New("localization bundle is nil")
		}
		if o.lcBundle != nil {
			return errors.New("localization bundle is already set")
		}
		o.lcBundle = b
		return nil
	}
}

func WithLogger(logger *slog.Logger) Option {
	return func(o *options) error {
		if logger == nil {
			return errors.New("logger is nil")
		}
		if o.logger != nil {
			return errors.New("logger is already set")
		}
		o.logger = logger
		return nil
	}
}

func requireAtLeastOneModelInStack(o *options) error {
	if len(o.stack) == 0 {
		return errors.New("cannot create a window without any initial models")
	}
	return nil
}
