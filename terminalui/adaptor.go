package terminalui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea/v2"
)

type domainAdaptor[T any] struct {
	Call        func(context.Context, T) (tea.Msg, error)
	CallContext context.Context
	TimeLimit   time.Duration
	Wrap        tea.Model
}

func NewDomainAdaptor[T any](
	callContext context.Context,
	call func(context.Context, T) (tea.Msg, error),
	wrap tea.Model,
) domainAdaptor[T] {
	if call == nil {
		panic("cannot use a <nil> domain call")
	}
	return domainAdaptor[T]{
		Call:        call,
		CallContext: callContext,
		TimeLimit:   time.Second * 20,
		Wrap:        wrap,
	}
}

func (d domainAdaptor[T]) Init() (_ tea.Model, cmd tea.Cmd) {
	d.Wrap, cmd = d.Wrap.Init()
	return d, cmd
}

func (d domainAdaptor[T]) Update(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	switch msg := msg.(type) {
	case T:
		return d, WithBusySignal(func() tea.Msg {
			ctx, cancel := context.WithTimeout(d.CallContext, d.TimeLimit)
			defer cancel()
			msg, err := d.Call(ctx, msg)
			if err != nil {
				return ErrorEvent(err)
			}
			return msg
		})
	}
	d.Wrap, cmd = d.Wrap.Update(msg)
	return d, cmd
}

func (d domainAdaptor[T]) View() string {
	return d.Wrap.View()
}
