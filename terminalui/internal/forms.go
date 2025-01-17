package internal

import (
	"reflect"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/dkotik/cuebook/patch"
	"github.com/dkotik/cuebook/terminalui/window"
)

func NewPatchCloser(name string) (func(tea.Model) tea.Model, func(patch.Patch) patchCloserCaptureRequest) {
	return func(m tea.Model) tea.Model {
			return patchCloser{Model: m, name: name}
		}, func(p patch.Patch) patchCloserCaptureRequest {
			return patchCloserCaptureRequest{name: name, patch: p}
		}
}

type patchCloserCaptureRequest struct {
	name  string
	patch patch.Patch
}

type patchCloser struct {
	tea.Model

	name      string
	lastPatch patch.Patch
}

func (p patchCloser) Init() (_ tea.Model, cmd tea.Cmd) {
	p.Model, cmd = p.Model.Init()
	return p, cmd
}

func (p patchCloser) Update(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	switch msg := msg.(type) {
	case patch.Result:
		if msg.LastChange != nil && reflect.DeepEqual(msg.LastChange, p.lastPatch) {
			return p.Model, func() tea.Msg { return window.BackEvent{} }
		}
	case patchCloserCaptureRequest:
		if msg.name == p.name {
			p.lastPatch = msg.patch
			return p, func() tea.Msg { return p.lastPatch } // unwrap patch
		}
		p.Model, cmd = p.Model.Update(msg)
		return p, cmd
	}
	p.Model, cmd = p.Model.Update(msg)
	return p, cmd
}
