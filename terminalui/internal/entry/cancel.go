package entry

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	listForm "github.com/dkotik/cuebook/terminalui/form"
	"github.com/dkotik/cuebook/terminalui/list"
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
	return listForm.NewBlankResponsiveLabel(list.NewButton(cancelButtonText, func() tea.Msg { return window.BackEvent{} }))
}
