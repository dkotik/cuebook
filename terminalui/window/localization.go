package window

import (
	"errors"
	"unicode"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

type TranslatableMessage interface {
	Translate(*i18n.Localizer) string
}

type Translatable interface {
	Translate(*i18n.Localizer) tea.Cmd
}

type TranslatableFunc func(*i18n.Localizer) tea.Cmd

func (t TranslatableFunc) Translate(lc *i18n.Localizer) tea.Cmd {
	return t(lc)
}

func (w window) issueLocalizedFlashErrorMessage(err error) (tea.Model, tea.Cmd) {
	return w, func() tea.Msg {
		var msg string
		var translatable TranslatableMessage
		if errors.As(err, &translatable) && w.localizer != nil {
			msg = translatable.Translate(w.localizer)
		} else {
			msg = err.Error()
		}
		// if msg == "" {
		// 	return nil
		// }
		// msg[0] =
		return flashMessageTemplate{
			Kind: FlashMessageKindError,
			Text: string(unicode.ToUpper(rune(msg[0]))) + msg[1:] + ".",
		}
		// return NewFlashMessage(
		// 	FlashMessageKindError,
		// 	&i18n.LocalizeConfig{
		// 		DefaultMessage: &i18n.Message{
		// 			ID: "windowFlashErrorMessage",
		// 			Other: "{{.Message}}.",
		// 		},
		// 		TemplateData: map[string]any{
		// 			"Message": msg,
		// 		},
		// 	},
		// )()
	}
}
