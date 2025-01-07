package terminalui

import tea "github.com/charmbracelet/bubbletea/v2"

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

func (l *loader[T]) Init() (tea.Model, tea.Cmd) {
	return l, tea.RequestWindowSize()
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
		var cmd tea.Cmd
		l.Model, cmd = msg.constructed.Init()
		return l, tea.Batch(tea.RequestWindowSize(), cmd)
	}

	var cmd tea.Cmd
	if l.Model != nil {
		l.Model, cmd = l.Model.Update(msg)
	}
	return l, cmd
}

func (l *loader[T]) View() string {
	if l.Model == nil {
		// TODO: flash screen?
		return "<loading>"
	}
	return l.Model.View()
}
