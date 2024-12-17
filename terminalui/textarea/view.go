package textarea

import (
	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/lipgloss/v2"
)

func (t Textarea) View() string {
	ta := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(55)).
		Padding(0).
		// Width(m.width).
		// Height(m.height-lipgloss.Height(header)-lipgloss.Height(footer)).
		// Align(lipgloss.Center, lipgloss.Center).
		Render(t.textarea.View())

	return lipgloss.JoinVertical(lipgloss.Left,
		" "+t.Label+":",
		ta, " "+t.help.ShortHelpView([]key.Binding{
			t.saveKey, t.escapeKey,
		}))
}
