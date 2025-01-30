package window

import (
	"errors"
	"fmt"
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

type TranslatableModel interface {
	tea.Model
	Translate(*i18n.Localizer) (TranslatableModel, error)
}

func NewTranslatableModel(m TranslatableModel) TranslatableModel {
	return translatableModel{TranslatableModel: m}
}

type translatableModel struct {
	TranslatableModel
}

func (t translatableModel) Init() (tea.Model, tea.Cmd) {
	model, cmd := t.TranslatableModel.Init()
	var ok bool
	t.TranslatableModel, ok = model.(TranslatableModel)
	if !ok {
		return model, tea.Batch(cmd, func() tea.Msg { return fmt.Errorf("model of type %T is not translatable", model) })
	}
	return t, cmd
}

func (t translatableModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	lc, ok := msg.(*i18n.Localizer)
	if ok {
		model, err := t.TranslatableModel.Translate(lc)
		if err != nil {
			return t, func() tea.Msg { return err }
		}
		t.TranslatableModel = model
	}
	model, cmd := t.TranslatableModel.Update(msg)

	t.TranslatableModel, ok = model.(TranslatableModel)
	if !ok {
		return model, tea.Batch(cmd, func() tea.Msg { return fmt.Errorf("model of type %T is not translatable", model) })
	}
	return t, cmd
}
