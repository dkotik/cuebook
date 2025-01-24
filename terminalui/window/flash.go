package window

import (
	"time"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

type FlashMessageKind uint8

type flashMessageTemplate struct {
	Kind FlashMessageKind
	Text string
}

type flashMessageClear struct{}

func (w window) ClearFlashMessageIfNeeded() window {
	if w.flashMessage != nil && time.Now().After(w.flashMessage.Expires) {
		w.flashMessage = nil
		w.flashMessageTemplate = nil
	}
	return w
}

func (w window) RenderFlashMessage(t flashMessageTemplate) tea.Cmd {
	style, ok := w.flashMessageStyles[t.Kind]
	if !ok {
		style = w.flashMessageStyles[FlashMessageKindError]
	}
	style = style.Width(max(0, w.size.Width-2))
	expires := time.Now().Add(w.FlashLingerDuration)
	return tea.Batch(
		func() tea.Msg {
			var m flashMessage
			m.Prefix = style.Render(string(t.Kind.Prefix()) + " ")
			m.Lines = style.Render(t.Text)
			m.Height = lipgloss.Height(m.Lines)
			m.Expires = expires
			return m
		},
		tea.Tick(w.FlashLingerDuration+time.Millisecond*100, func(_ time.Time) tea.Msg {
			return flashMessageClear{}
		}),
	)
}

type flashMessage struct {
	Prefix  string
	Lines   string
	Height  int
	Expires time.Time
}

const (
	FlashMessageKindInfo FlashMessageKind = iota
	FlashMessageKindSuccess
	FlashMessageKindWarning
	FlashMessageKindError
)

const (
	flashPrefixSuccess = '✔'
	flashPrefixInfo    = 'ⓘ'
	flashPrefixWarning = '💥' // ✱
	flashPrefixError   = '✖' // 💥
)

func (k FlashMessageKind) Prefix() rune {
	switch k {
	case FlashMessageKindSuccess:
		return flashPrefixSuccess
	case FlashMessageKindWarning:
		return flashPrefixWarning
	case FlashMessageKindError:
		return flashPrefixError
	default:
		return flashPrefixInfo
	}
}

func NewFlashMessage(kind FlashMessageKind, text *i18n.LocalizeConfig) tea.Cmd {
	return func() tea.Msg {
		return TranslatableFunc(func(lc *i18n.Localizer) tea.Cmd {
			return func() tea.Msg {
				localized, err := lc.Localize(text)
				if err != nil {
					return err
				}
				return flashMessageTemplate{
					Kind: kind,
					Text: localized,
				}
			}
		})
	}
}
