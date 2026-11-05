package window

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func (w window) View() tea.View {
	view := w.current.View()
	if w.flashMessage != nil {
		view.SetContent(lipgloss.JoinVertical(
			lipgloss.Left, // TODO: margin should be determined by the window not the list component
			view.Content,
			lipgloss.JoinHorizontal( // TODO: take FlashMaximumHeight into account
				lipgloss.Top,
				" "+w.flashMessage.Prefix+" ",
				w.flashMessage.Lines,
			),
		))
	}
	view.AltScreen = true
	view.MouseMode = tea.MouseModeCellMotion
	return view
}
