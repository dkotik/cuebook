package file

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/bubbles/v2/filepicker"
	tea "github.com/charmbracelet/bubbletea/v2"
)

type Command func(filePath string) tea.Cmd

type file struct {
	filepicker.Model

	OnSelect Command
	// Path     string
}

func New(withOptions ...Option) (_ tea.Model, err error) {
	o := &options{}
	for _, option := range append(
		withOptions,
		func(o *options) error {
			if len(o.AllowedExtensions) == 0 {
				return errors.New("no file extensions provided")
			}
			if o.OnSelect == nil {
				o.OnSelect = func(filePath string) tea.Cmd {
					return func() tea.Msg {
						return LoadEvent(filePath)
					}
				}
			}
			return nil
		},
	) {
		if err = option(o); err != nil {
			return nil, fmt.Errorf("cannot create file browser: %w", err)
		}
	}
	fp := filepicker.New()
	fp.AllowedTypes = o.AllowedExtensions
	// fp.CurrentDirectory, _ = os.Getwd()
	fp.AutoHeight = true
	fp.DirAllowed = false
	fp.FileAllowed = true
	fp.ShowPermissions = false
	fp.ShowSize = false
	fp.ShowHidden = true
	return file{
		Model:    fp,
		OnSelect: o.OnSelect,
	}, nil
}

func (f file) Load() (tea.Model, tea.Cmd) {
	content, err := os.ReadFile(f.Path)
	if err != nil {
		return f, func() tea.Msg {
			return err
		}
	}
	return f, func() tea.Msg {
		return ContentEvent(content)
	}
}

func (f file) Init() (_ tea.Model, cmd tea.Cmd) {
	f.Model, cmd = f.Model.Init()
	// if f.Path != "" {
	// 	return f.Load() // TODO: skipping cmd!
	// }
	return f, cmd
}

func (f file) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Key().Code {
		case tea.KeyEnter:
			var cmd tea.Cmd
			f.Model, cmd = f.Model.Update(msg)
			ok, filePath := f.Model.DidSelectFile(msg)
			if !ok {
				return f, cmd
			}
			return f, tea.Batch(f.OnSelect(filePath), cmd)
		case tea.KeyF5:
			return f.Load()
		}
	case LoadEvent:
		f.Path = string(msg)
		return f, func() tea.Msg {
			content, err := os.ReadFile(string(msg))
			if err != nil {
				return err
			}
			return ContentEvent(content)
		}
	case saveEvent:
		return f, func() tea.Msg {
			// TODO: first create a hashed file, then overwrite the target
			handle, err := os.Create(f.Path)
			if err != nil {
				return err
			}
			if _, err = io.Copy(handle, bytes.NewReader([]byte(msg))); err != nil {
				return errors.Join(err, handle.Close())
			}
			if err = handle.Close(); err != nil {
				return err
			}
			return UpdateEvent(msg)
		}
	}
	var cmd tea.Cmd
	f.Model, cmd = f.Model.Update(msg)
	return f, cmd
}
