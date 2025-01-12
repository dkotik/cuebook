package window

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/dkotik/cuebook/terminalui/event"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"

	tea "github.com/charmbracelet/bubbletea/v2"
)

func New(withOptions ...Option) (_ tea.Model, err error) {
	o := &options{
		stack: make([]tea.Model, 0, 5),
	}
	for _, option := range append(
		withOptions,
		func(o *options) error { // validate options
			if len(o.stack) == 0 {
				return errors.New("cannot create a window without any initial models")
			}
			return nil
		},
	) {
		if err = option(o); err != nil {
			return nil, fmt.Errorf("unable to create terminal window component: %w", err)
		}
	}

	lastModel := len(o.stack) - 1
	return window{
		commandContext: cmp.Or(o.commandContext, context.Background()),
		current:        o.stack[lastModel],
		stack:          o.stack[:lastModel],
		watchers:       o.watchers,
		lcBundle:       cmp.Or(o.lcBundle, i18n.NewBundle(language.AmericanEnglish)),
		logger:         cmp.Or(o.logger, slog.Default()),
	}, nil
}

type window struct {
	commandContext context.Context
	current        tea.Model
	stack          []tea.Model
	watchers       []tea.Model
	size           tea.WindowSizeMsg
	busy           uint8
	lcBundle       *i18n.Bundle
	localizer      *i18n.Localizer
	logger         *slog.Logger
}

func (w window) Init() (_ tea.Model, cmd tea.Cmd) {
	all := make([]tea.Cmd, 0, len(w.stack)+len(w.watchers)+1)
	for i, model := range w.watchers {
		w.watchers[i], cmd = model.Init()
		all = append(all, cmd)
	}
	for i, model := range w.stack {
		w.stack[i], cmd = model.Init()
		all = append(all, cmd)
	}
	w.current, cmd = w.current.Init()
	all = append(all, cmd)
	return w, tea.Batch(all...)
}

func (w window) back() (tea.Model, tea.Cmd) {
	if l := len(w.stack); l > 0 {
		l -= 1
		w.current = w.stack[l]
		w.stack = w.stack[:l]
		var cmd tea.Cmd
		w.current, cmd = w.current.Update(w.size)
		return w, cmd
	}
	return w, tea.Quit
}

func (w window) Update(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	w.logger.Debug(spew.Sdump(msg))
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		w.size = msg
	case *i18n.Localizer:
		if msg == nil {
			panic("nil localizer")
		}
		w.localizer = msg
	case commandContextRequestEvent:
		return w, func() tea.Msg { return w.commandContext }
	case localizerRequestEvent:
		localizer := w.localizer
		return w, func() tea.Msg { return localizer }
	case SwitchTo:
		w.stack = append(w.stack, w.current)
		var cmdInit tea.Cmd
		w.current, cmdInit = msg.Init()
		w.current, cmd = w.current.Update(w.size) // TODO: should be requested instead of fed, like localizer
		return w, tea.Batch(cmd, cmdInit)
	case BackEvent:
		return w.back()
	case SetBusyEvent:
		if bool(msg) {
			w.busy++
			return w, func() tea.Msg {
				return IsBusyEvent(true)
			}
		} else {
			w.busy--
			if w.busy == 0 {
				return w, func() tea.Msg {
					// delay busy reset
					// to allow the animation to play longer
					time.Sleep(time.Second)
					return IsBusyEvent(false)
				}
			}
		}
		return w, nil
	case tea.KeyMsg:
		if w.busy != 0 {
			return w, nil // drop event if busy
		}
		switch msg.Key().Code {
		case tea.KeyEscape:
			return w.back()
		}
		// fallthrough
		w.current, cmd = w.current.Update(msg)
		return w, tea.Batch(
			cmd,
			event.Propagate(msg, w.watchers),
		)
	case tea.MouseMsg, tea.CursorPositionMsg, tea.PasteStartMsg, tea.PasteEndMsg, tea.PasteMsg: // input events go to current model only
		if w.busy != 0 {
			return w, nil // drop event if busy
		}
		w.current, cmd = w.current.Update(msg)
		return w, tea.Batch(
			cmd,
			event.Propagate(msg, w.watchers),
		)
	}
	w.current, cmd = w.current.Update(msg)
	return w, tea.Batch(
		cmd,
		event.Propagate(msg, w.stack),
		event.Propagate(msg, w.watchers),
	)
}

func (w window) View() string {
	return w.current.View()
}
