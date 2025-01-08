package internal

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/dkotik/cuebook"
)

func New(model tea.Model) tea.Model {
	if model == nil {
		panic("state cannot track a nil model")
	}
	return state{Model: model}
}

type state struct {
	tea.Model

	EntryCount int
	FieldCount int
	Book       cuebook.CueBook
	Source     []byte
}

func (s state) IsReady() bool {
	return len(s.Source) > 0
}

func (s state) Init() (_ tea.Model, cmd tea.Cmd) {
	s.Model, cmd = s.Model.Init()
	return s, cmd
}
