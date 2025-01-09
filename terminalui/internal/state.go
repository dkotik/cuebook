package internal

import (
	"cmp"
	"context"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/dkotik/cuebook"
)

func NewWithCueState(ctx context.Context, model tea.Model) tea.Model {
	if model == nil {
		panic("state cannot track a nil model")
	}
	return state{
		Model:   model,
		Context: cmp.Or(ctx, context.Background()),
	}
}

type state struct {
	tea.Model
	Context            context.Context
	SelectedEntryIndex int
	SelectedFieldIndex int
	Book               cuebook.CueBook
	Source             []byte
}

func (s state) IsReady() bool {
	return len(s.Source) > 0
}

func (s state) IsEntryListAvailable() bool {
	return s.SelectedEntryIndex != -2
}

func (s state) IsFieldListAvailable() bool {
	return s.SelectedFieldIndex != -2
}

func (s state) Init() (_ tea.Model, cmd tea.Cmd) {
	s.Model, cmd = s.Model.Init()
	s.SelectedEntryIndex = -2
	s.SelectedFieldIndex = -2
	return s, cmd
}
