package window

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

type options struct {
	commandContext      context.Context
	stack               []tea.Model
	watchers            []tea.Model
	lcBundle            *i18n.Bundle
	logger              *slog.Logger
	FlashMaximumHeight  int
	FlashLingerDuration time.Duration
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

func WithFlashMaximumHeight(h int) Option {
	return func(o *options) error {
		if h < 1 {
			return errors.New("maximum height must be at least 1")
		}
		if o.FlashMaximumHeight != 0 {
			return errors.New("maximum height is already set")
		}
		o.FlashMaximumHeight = h
		return nil
	}
}

func WithFlashMessageLingerDuration(d time.Duration) Option {
	return func(o *options) error {
		if d < time.Millisecond*50 {
			return errors.New("linger duration must be at least 50ms")
		}
		if o.FlashLingerDuration != 0 {
			return errors.New("linger duration is already set")
		}
		o.FlashLingerDuration = d
		return nil
	}
}
