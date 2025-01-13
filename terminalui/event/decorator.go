package event

import tea "github.com/charmbracelet/bubbletea/v2"

type (
	Middleware func(tea.Cmd) tea.Cmd
	Decorator  func(tea.Model) tea.Model
)

func NewDecorator(mw Middleware) Decorator {
	if mw == nil {
		panic("middleware is nil")
	}
	return func(model tea.Model) tea.Model {
		if model == nil {
			panic("decorated model is nil")
		}
		return decorator{
			Model:      model,
			Middleware: mw,
		}
	}
}

type decorator struct {
	tea.Model
	Middleware
}

func (d decorator) Init() (_ tea.Model, cmd tea.Cmd) {
	d.Model, cmd = d.Model.Init()
	if cmd == nil {
		return d, nil
	}
	return d, d.Middleware(cmd)
}

func (d decorator) Update(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	d.Model, cmd = d.Model.Update(msg)
	if cmd == nil {
		return d, nil
	}
	return d, d.Middleware(cmd)
}
