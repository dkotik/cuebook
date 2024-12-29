package list

import (
	"iter"
	"slices"
	"strings"

	"github.com/charmbracelet/lipgloss/v2"
)

func (l List) cardSeparator() string {
	return strings.Repeat(" ", l.Size.Width)
}

func (l List) forwardLineIterator() iter.Seq[string] {
	separator := l.cardSeparator()
	return func(yield func(value string) bool) {
		for _, card := range l.Items[l.SelectedIndex+1:] {
			// TODO: do not return the last separator, check if final item
			if !yield(separator) {
				return
			}

			normalized := strings.ReplaceAll(card.View(), "\r\n", "\n") // normalize line endings
			for _, line := range strings.Split(normalized, "\n") {
				if !yield(line) {
					return
				}
			}
		}
	}
}

func (l List) reverseLineIterator() iter.Seq[string] {
	separator := l.cardSeparator()
	return func(yield func(value string) bool) {
		for i := l.SelectedIndex - 1; i >= 0; i-- {
			if !yield(separator) {
				return
			}

			normalized := strings.ReplaceAll(l.Items[i].View(), "\r\n", "\n") // normalize line endings
			lines := strings.Split(normalized, "\n")
			for j := len(lines) - 1; j >= 0; j-- {
				if !yield(lines[j]) {
					return
				}
			}
		}
	}
}

func (l List) View() string {
	total := len(l.Items)
	if total == 0 {
		return "</>"
	}

	view := l.Items[l.SelectedIndex].View()
	space := l.Size.Height - lipgloss.Height(view)
	if space < 0 {
		l.fullScreenView.SetContent(view)
		view = l.fullScreenView.View()
	} else if space == 0 { // exact fit
		return view
	} else {
		below, stopBelow := iter.Pull(l.forwardLineIterator())
		defer stopBelow()
		above, stopAbove := iter.Pull(l.reverseLineIterator())
		defer stopAbove()

		var (
			lines = make([]string, 0, l.Size.Height)
			line  string
			ok    bool
		)

		for range space / 2 { // above selected item
			line, ok = above()
			if !ok {
				break
			}
			lines = append(lines, line)
			space--
		}
		slices.Reverse(lines)
		lines = append(lines, view)

		for range space { // below selected item
			line, ok = below()
			if !ok {
				break
			}
			lines = append(lines, line)
			space--
		}

		for range space { // fill any space available above
			line, ok = above()
			if !ok {
				break
			}
			lines = append([]string{line}, lines...)
			space--
		}
		view = lipgloss.JoinVertical(lipgloss.Left, lines...)
	}

	return lipgloss.NewStyle().
		// AlignHorizontal(lipgloss.Left).
		AlignVertical(lipgloss.Center).
		Height(l.Size.Height).
		MarginLeft(l.Size.Width / 4).
		// MarginRight(l.Size.Width / 3).
		Render(view)
}
