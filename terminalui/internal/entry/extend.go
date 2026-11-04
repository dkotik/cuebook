package entry

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/dkotik/cuebook/terminalui/internal/label"
	"github.com/dkotik/cuebook/terminalui/internal/list"
	"github.com/dkotik/cuebook/terminalui/internal/textarea"
	"github.com/dkotik/cuebook/terminalui/window"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

type extendEvent struct {
	Name string
}

func NewExtendButton() tea.Model {
	return label.New(list.NewButton(
		&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "bookEntryExtend",
				Other: "Add a Custom Field",
			},
		},
		func() tea.Msg {
			ta, err := textarea.New(textarea.Configuration{
				Label: "Add a Custom Field", // TODO: localize
				OnSubmit: func(value string) tea.Cmd {
					return func() tea.Msg {
						return extendEvent{
							Name: value,
						}
					}
				},
			})
			if err != nil {
				return err
			}
			return window.SwitchTo(ta)
		},
	), nil)
}
