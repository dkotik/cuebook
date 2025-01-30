package internal

import (
	tea "github.com/charmbracelet/bubbletea/v2"
)

type flashAnnouncer struct {
	FileName string // TODO: populate from file selector signal, include in flash message
}

func (f flashAnnouncer) Init() (_ tea.Model, cmd tea.Cmd) {
	return f, nil
}

func (f flashAnnouncer) Update(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	// switch msg := msg.(type) {
	// case patch.Result:
	//
	//	switch patch := msg.LastChange.(type) {
	//	// TODO: handle other patch types: swapEntriesPatch, swapFieldsPatch
	//	case updateFieldPatch:
	//		return f, window.NewFlashMessage(window.FlashMessageKindSuccess, &i18n.LocalizeConfig{
	//			DefaultMessage: &i18n.Message{
	//				ID:    "flashMessageEntryUpdated",
	//				Other: "Entry `{{.Title}}` updated.",
	//			},
	//			TemplateData: map[string]interface{}{
	//				"Title": patch.Entry.GetTitle(),
	//			},
	//		})
	//	case deleteEntryPatch:
	//		return f, window.NewFlashMessage(window.FlashMessageKindWarning, &i18n.LocalizeConfig{
	//			DefaultMessage: &i18n.Message{
	//				ID:    "flashMessageEntryDeleted",
	//				Other: "Entry `{{.Title}}` deleted.",
	//			},
	//			TemplateData: map[string]interface{}{
	//				"Title": patch.Entry.GetTitle(),
	//			},
	//		})
	//	default:
	//		return f, nil
	//	}
	//
	// default:
	//
	//		return f, nil
	//	}
	return f, nil
}

func (f flashAnnouncer) View() string {
	return "<flash announcer>"
}
