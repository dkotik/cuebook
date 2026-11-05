package status

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

func (m model) View() tea.View {
	s := strings.Builder{}
	if m.Busy {
		_, _ = s.WriteString(m.Spinner.View())
	} else {
		_, _ = s.WriteRune(' ')
	}

	_, _ = s.WriteRune(' ')
	_, _ = s.WriteString(m.Help.ShortHelpView(m.KeyBindings))

	_, _ = s.WriteString(fmt.Sprintf(" %v", m.Busy))

	return tea.NewView(s.String())
}
