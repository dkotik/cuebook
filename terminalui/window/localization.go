package window

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

type Translatable interface {
	Translate(*i18n.Localizer) tea.Cmd
}

type TranslatableFunc func(*i18n.Localizer) tea.Cmd

func (t TranslatableFunc) Translate(lc *i18n.Localizer) tea.Cmd {
	return t(lc)
}
