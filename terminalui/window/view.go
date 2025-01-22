package window

import "github.com/charmbracelet/lipgloss/v2"

func (w window) View() string {
	if w.flashMessage == nil {
		return w.current.View()
	}
	return lipgloss.JoinVertical(
		lipgloss.Left, // TODO: margin should be determined by the window not the list component
		w.current.View(),
		lipgloss.JoinHorizontal( // TODO: take FlashMaximumHeight into account
			lipgloss.Top,
			w.flashMessage.Prefix+" ",
			w.flashMessage.Lines,
		),
	)
}
