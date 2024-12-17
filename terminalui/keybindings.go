package terminalui

import (
	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func NewSaveKey(lc *i18n.Localizer) key.Binding {
	return key.NewBinding(
		key.WithKeys("ctrl+s", "cmd+s", "alt-s"),
		key.WithHelp("ctrl+s", lc.MustLocalize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "cuebook.keyBinding.save",
				Other: "save",
			},
		})),
	)
}

func NewCancelKey(lc *i18n.Localizer) key.Binding {
	return key.NewBinding(
		key.WithKeys("escape"),
		key.WithHelp("esc", lc.MustLocalize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "cuebook.keyBinding.cancel",
				Other: "cancel",
			},
		})),
	)
}

func NewQuitKey(lc *i18n.Localizer) key.Binding {
	return key.NewBinding(
		key.WithKeys("escape"),
		key.WithHelp("esc", lc.MustLocalize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "cuebook.keyBinding.quit",
				Other: "quit",
			},
		})),
	)
}
