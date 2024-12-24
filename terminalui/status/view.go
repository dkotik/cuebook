package status

import (
	"strings"
)

func (m model) View() string {
	s := strings.Builder{}
	if m.Busy {
		_, _ = s.WriteString(m.Spinner.View())
	} else {
		_, _ = s.WriteRune(' ')
	}

	_, _ = s.WriteRune(' ')
	_, _ = s.WriteString(m.Help.ShortHelpView(m.KeyBindings))

	return s.String()
}
