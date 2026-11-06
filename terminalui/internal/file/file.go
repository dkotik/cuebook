package file

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"

	"charm.land/bubbles/v2/filepicker"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type Command func(filePath string) tea.Cmd

type file struct {
	filepicker.Model

	OnSelect Command
	// Path     string
}

// Configuration controls file selection and appearance. Unset style properties
// inherit the [filepicker.DefaultStyles]. A nil OnSelect emits a [LoadEvent].
type Configuration struct {
	AllowedExtensions []string
	OnSelect          Command

	CursorStyle           lipgloss.Style
	DisabledCursorStyle   lipgloss.Style
	DirectoryStyle        lipgloss.Style
	SymlinkStyle          lipgloss.Style
	FileStyle             lipgloss.Style
	DisabledFileStyle     lipgloss.Style
	SelectedStyle         lipgloss.Style
	DisabledSelectedStyle lipgloss.Style
	EmptyDirectoryStyle   lipgloss.Style
}

func New(c Configuration) (_ tea.Model, err error) {
	if err = c.validateAllowedExtensions(); err != nil {
		return nil, fmt.Errorf("cannot create file browser: %w", err)
	}
	if c.OnSelect == nil {
		c.OnSelect = func(filePath string) tea.Cmd {
			return func() tea.Msg {
				return LoadEvent(filePath)
			}
		}
	}

	fp := filepicker.New()
	fp.AllowedTypes = c.AllowedExtensions
	fp.Styles = c.styles()
	// fp.CurrentDirectory, _ = os.Getwd()
	fp.AutoHeight = true
	fp.DirAllowed = false
	fp.FileAllowed = true
	fp.ShowPermissions = false
	fp.ShowSize = false
	fp.ShowHidden = true
	return file{
		Model:    fp,
		OnSelect: c.OnSelect,
	}, nil
}

func (c *Configuration) validateAllowedExtensions() error {
	if len(c.AllowedExtensions) == 0 {
		return errors.New("no file extensions provided")
	}
	unique := make([]string, 0, len(c.AllowedExtensions))
	for _, ext := range c.AllowedExtensions {
		if ext == "" {
			return errors.New("empty file extension provided")
		}
		if ext[0] != '.' {
			return errors.New("a file extension must begin with a dot")
		}
		if !slices.Contains(unique, ext) {
			unique = append(unique, ext)
		}
	}
	c.AllowedExtensions = unique
	return nil
}

func (c Configuration) styles() filepicker.Styles {
	d := filepicker.DefaultStyles()
	d.Cursor = c.CursorStyle.Inherit(d.Cursor)
	d.DisabledCursor = c.DisabledCursorStyle.Inherit(d.DisabledCursor)
	d.Directory = c.DirectoryStyle.Inherit(d.Directory)
	d.Symlink = c.SymlinkStyle.Inherit(d.Symlink)
	d.File = c.FileStyle.Inherit(d.File)
	d.DisabledFile = c.DisabledFileStyle.Inherit(d.DisabledFile)
	d.Selected = c.SelectedStyle.Inherit(d.Selected)
	d.DisabledSelected = c.DisabledSelectedStyle.Inherit(d.DisabledSelected)
	// Inherit skips padding and the message text, so restore them unless set.
	empty := c.EmptyDirectoryStyle.Inherit(d.EmptyDirectory)
	if c.EmptyDirectoryStyle.GetPaddingLeft() == 0 {
		empty = empty.PaddingLeft(d.EmptyDirectory.GetPaddingLeft())
	}
	if c.EmptyDirectoryStyle.Value() == "" {
		empty = empty.SetString(d.EmptyDirectory.Value())
	}
	d.EmptyDirectory = empty
	return d
}

func (f file) Load() (tea.Model, tea.Cmd) {
	content, err := os.ReadFile(f.Path)
	if err != nil {
		return f, func() tea.Msg {
			return err
		}
	}
	return f, func() tea.Msg {
		return ContentEvent{Path: f.Path, Content: content}
	}
}

func (f file) Init() tea.Cmd {
	return f.Model.Init()
}

func (f file) View() tea.View {
	return tea.NewView(f.Model.View())
}

func (f file) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.Key().Code {
		case tea.KeyEsc:
			cwd, _ := os.Getwd()
			picked := filepath.Join(cwd, f.Model.CurrentDirectory)
			if picked == cwd {
				return f, nil // allows window escape handler to navigate back
			}
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
			return ContentEvent{Path: string(msg), Content: content}
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
			return UpdateEvent{Path: f.Path, Content: []byte(msg)}
		}
	}
	var cmd tea.Cmd
	f.Model, cmd = f.Model.Update(msg)
	return f, cmd
}

// // HandleEscapeKey satisfies [window.EscapeKeyHandler] interface
// // to block navigation away from the window
// func (f file) HandleEscapeKey(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
// 	cwd, _ := os.Getwd()
// 	if f.Model.Path == cwd {
// 		panic("end")
// 	}
// 	f.Model, cmd = f.Model.Update(msg)
// 	return f, cmd
// }
