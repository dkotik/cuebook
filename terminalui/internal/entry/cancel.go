package entry

import (
	tea "charm.land/bubbletea/v2"
	"github.com/dkotik/cuebook/terminalui/internal/label"
	"github.com/dkotik/cuebook/terminalui/internal/list"
	"github.com/dkotik/cuebook/terminalui/window"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

var cancelButtonText = &i18n.LocalizeConfig{
	DefaultMessage: &i18n.Message{
		ID:    "bookEntryDiscardUnsavedChanges",
		Other: "Cancel",
	},
}

type cancelEvent struct{}

func NewCancelButton() tea.Model {
	return label.New(list.NewButton(cancelButtonText, func() tea.Msg { return window.BackEvent{} }), nil)
}
