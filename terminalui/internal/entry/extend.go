package entry

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	listForm "github.com/dkotik/cuebook/terminalui/form"
	"github.com/dkotik/cuebook/terminalui/list"
	"github.com/dkotik/cuebook/terminalui/textarea"
	"github.com/dkotik/cuebook/terminalui/window"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

type extendEvent struct {
	Name string
}

func NewExtendButton() tea.Model {
	return listForm.NewBlankResponsiveLabel(list.NewButton(
		&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "bookEntryExtend",
				Other: "Add a Custom Field",
			},
		},
		func() tea.Msg {
			ta, err := textarea.New(
				textarea.WithLabel("Add a Custom Field"), // TODO: localize
				textarea.WithOnSubmitCommand(func(value string) tea.Cmd {
					return func() tea.Msg {
						return extendEvent{
							Name: value,
						}
					}
				}),
			)
			if err != nil {
				return err
			}
			return window.SwitchTo(ta)
		},
	))
}
