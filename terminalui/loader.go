package terminalui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/dkotik/cuebook/terminalui/window"
)

type ModelConstructor[T any] func(T) (tea.Model, error)

// NewLoader creates a model that initiates and refreshes itself when receiving a certain event type.
func NewLoader[T any](constructor ModelConstructor[T]) tea.Model {
	// note pointer reciever for detecting loadingCompleteEvent created by self later
	return &loader[T]{
		modelConstructor: constructor,
	}
}

type loadingCompleteEvent struct {
	loader      tea.Model
	constructed tea.Model
}

type loader[T any] struct {
	tea.Model
	modelConstructor ModelConstructor[T]
}

func (l *loader[T]) Init() tea.Cmd {
	return window.RequestWindowSize()
}

func (l *loader[T]) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case T:
		return l, func() tea.Msg {
			m, err := l.modelConstructor(msg)
			if err != nil {
				return err
			}
			return loadingCompleteEvent{
				loader:      l,
				constructed: m,
			}
		}
	case loadingCompleteEvent:
		if msg.loader != l {
			// panic("another loader triggered loading complete")
			return l, nil
		}
		l.Model = msg.constructed
		cmd := l.Model.Init()
		return l, tea.Batch(window.RequestWindowSize(), cmd)
	}

	var cmd tea.Cmd
	if l.Model != nil {
		l.Model, cmd = l.Model.Update(msg)
	}
	return l, cmd
}

func (l *loader[T]) View() tea.View {
	if l.Model == nil {
		// TODO: flash screen?
		return tea.NewView("<loading>")
	}
	return l.Model.View()
}
