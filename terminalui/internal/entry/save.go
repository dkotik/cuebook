package entry

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/dkotik/cuebook/patch"
	listForm "github.com/dkotik/cuebook/terminalui/form"
	"github.com/dkotik/cuebook/terminalui/list"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// Update creates a source patch that modifies one entry.
type Update func(patch.Result) (patch.Patch, error)

func NewSaveButton() tea.Model {
	return listForm.NewBlankResponsiveLabel(list.NewButton(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "bookEntrySave",
			Other: "Save Changes",
		},
	}, func() tea.Msg { return listForm.SaveChangesEvent{} }))
}
