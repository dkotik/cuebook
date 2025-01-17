package textarea

import (
	"errors"
	"strings"

	tea "github.com/charmbracelet/bubbletea/v2"
)

type (
	Command func(string) tea.Cmd
	Option  func(*options) error
)

type options struct {
	Label string
	Value string
	// Required bool
	OnSubmit Command
}

func WithLabel(label string) Option {
	return func(o *options) error {
		if strings.TrimSpace(label) == "" {
			return errors.New("empty label")
		}
		if o.Label != "" {
			return errors.New("label is already set")
		}
		o.Label = label
		return nil
	}
}

func WithValue(value string) Option {
	return func(o *options) error {
		if strings.TrimSpace(value) == "" {
			return errors.New("empty value")
		}
		if o.Value != "" {
			return errors.New("value is already set")
		}
		o.Value = value
		return nil
	}
}

func WithOnSubmitCommand(cmd Command) Option {
	return func(o *options) error {
		if cmd == nil {
			return errors.New("empty command")
		}
		if o.OnSubmit != nil {
			return errors.New("command is already set")
		}
		o.OnSubmit = cmd
		return nil
	}
}
