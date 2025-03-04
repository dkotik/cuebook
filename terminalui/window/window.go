package window

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/dkotik/cuebook/terminalui/event"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
)

func New(withOptions ...Option) (_ tea.Model, err error) {
	o := &options{
		stack: make([]tea.Model, 0, 5),
	}
	for _, option := range append(
		withOptions,
		func(o *options) error { // validate options
			if o.FlashMaximumHeight == 0 {
				o.FlashMaximumHeight = 5
			}
			if o.FlashLingerDuration == 0 {
				o.FlashLingerDuration = time.Second * 3
			}
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
		flashMessageStyles: map[FlashMessageKind]lipgloss.Style{
			FlashMessageKindInfo:    lipgloss.NewStyle().Foreground(lipgloss.BrightCyan),
			FlashMessageKindSuccess: lipgloss.NewStyle().Foreground(lipgloss.BrightGreen),
			FlashMessageKindWarning: lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5F1F")), // Neon Orange
			FlashMessageKindError:   lipgloss.NewStyle().Foreground(lipgloss.BrightRed),
		},
		FlashLingerDuration: o.FlashLingerDuration,
		FlashMaximumHeight:  o.FlashMaximumHeight,
		lcBundle:            cmp.Or(o.lcBundle, i18n.NewBundle(language.AmericanEnglish)),
		logger:              cmp.Or(o.logger, slog.Default()),
	}, nil
}

type window struct {
	commandContext context.Context
	current        tea.Model
	stack          []tea.Model
	watchers       []tea.Model
	size           tea.WindowSizeMsg
	busy           uint8

	FlashLingerDuration  time.Duration
	FlashMaximumHeight   int
	flashMessageStyles   map[FlashMessageKind]lipgloss.Style
	flashMessageTemplate *flashMessageTemplate
	flashMessage         *flashMessage

	lcBundle  *i18n.Bundle
	localizer *i18n.Localizer
	logger    *slog.Logger
}

func (w window) Init() (_ tea.Model, cmd tea.Cmd) {
	w.localizer = i18n.NewLocalizer(w.lcBundle, "en")
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

func (w window) Update(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	w.logger.Debug(fmt.Sprintf("%T", msg), slog.Any("payload", msg))

	switch msg := msg.(type) {
	case error:
		w.current, cmd = w.current.Update(msg)
		if cmd == nil {
			return w.issueLocalizedFlashErrorMessage(msg)
		}
	case commandContextRequestEvent:
		return w, func() tea.Msg { return w.commandContext }
	case localizerRequestEvent:
		localizer := w.localizer
		return w, func() tea.Msg { return localizer }
		// cmd = func() tea.Msg { return localizer }
		// return w, tea.Batch(
		// 	event.Propagate(w.localizer, w.stack),
		// 	event.Propagate(w.localizer, w.watchers),
		// )
	case Translatable:
		return w, msg.Translate(w.localizer)
	case SwitchTo:
		w.stack = append(w.stack, w.current)
		var cmdInit tea.Cmd
		w.current, cmdInit = msg.Init()
		return w, cmdInit
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
		w.current, cmd = w.current.Update(msg)
		if cmd == nil && msg.Key().Code == tea.KeyEsc {
			return w.back()
		}
		return w.ClearFlashMessageIfNeeded(), tea.Batch(
			cmd,
			event.Propagate(msg, w.watchers),
		)
	case tea.MouseMsg, tea.CursorPositionMsg, tea.PasteStartMsg, tea.PasteEndMsg, tea.PasteMsg: // input events go to current model only
		if w.busy != 0 {
			return w, nil // drop event if busy
		}
		w.current, cmd = w.current.Update(msg)
		return w.ClearFlashMessageIfNeeded(), tea.Batch(
			cmd,
			event.Propagate(msg, w.watchers),
		)
	case flashMessageClear:
		w = w.ClearFlashMessageIfNeeded()
		if w.flashMessage == nil {
			return w, nil
		}
		// flash message was not yet ready to clear, so schedule another clearing attempt
		return w, tea.Tick(time.Until(w.flashMessage.Expires)+time.Microsecond*100, func(_ time.Time) tea.Msg {
			return flashMessageClear{}
		})
	case flashMessageTemplate:
		w.flashMessageTemplate = &msg
		return w, w.RenderFlashMessage(msg)
	case flashMessage:
		w.flashMessage = &msg
		return w, tea.RequestWindowSize()
	case tea.WindowSizeMsg:
		if w.flashMessage != nil {
			msg.Height = max(0, msg.Height-w.flashMessage.Height)
		}
	}
	w.current, cmd = w.current.Update(msg)
	return w, tea.Batch(
		cmd,
		event.Propagate(msg, w.stack),
		event.Propagate(msg, w.watchers),
	)
}
