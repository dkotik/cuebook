package file

import (
	"bytes"
	"errors"
	"io"
	"os"

	tea "github.com/charmbracelet/bubbletea/v2"
)

type file struct {
	Path string
}

func New(p string) tea.Model {
	return file{
		Path: p,
	}
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

func (f file) Init() (tea.Model, tea.Cmd) {
	if f.Path != "" {
		return f.Load()
	}
	return f, nil
}

func (f file) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Key().Code {
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
	case SaveEvent:
		return f, func() tea.Msg {
			// TODO: first create a hashed file, then overwrite the target
			handle, err := os.Create(string(msg))
			if err != nil {
				return err
			}
			if _, err = io.Copy(handle, bytes.NewReader([]byte(msg))); err != nil {
				return errors.Join(err, handle.Close())
			}
			if err = handle.Close(); err != nil {
				return err
			}
			return nil
		}
	}
	return f, nil
}

func (f file) View() string {
	return "<file picker>" // TODO: implement
}
