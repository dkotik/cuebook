package terminalui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/truncate"
)

type SwitchTo tea.Model

func NewSwitch(left, right tea.Model, reverse bool) tea.Model {
	return slideSwitch{
		Left:     left,
		Right:    right,
		Reverse:  reverse,
		Progress: 0.01,
	}
}

type advanceSlideAnimation struct{}

type slideSwitch struct {
	Left     tea.Model
	Right    tea.Model
	Reverse  bool
	Progress float32
	Size     tea.WindowSizeMsg
}

func (s slideSwitch) Init() (tea.Model, tea.Cmd) {
	return s, tea.Batch(
		tea.RequestWindowSize(),
		func() tea.Msg { return advanceSlideAnimation{} },
	)
}

func (s slideSwitch) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Key().Code == tea.KeyEscape {
			s.Reverse = true
			return s, nil
		}
	case tea.WindowSizeMsg:
		s.Size = msg
		s.Left, _ = s.Left.Update(msg) // TODO: propagadate cmd?
		s.Right, _ = s.Right.Update(msg)
		return s, nil
	case advanceSlideAnimation:
		if s.Size.Width > 0 {
			s.Progress *= 1.01
			if s.Progress > 1.0 {
				if s.Reverse {
					return s.Left.Update(s.Size)
				}
				return s.Right.Update(s.Size)
			}
		}
		return s, func() tea.Msg {
			<-time.After(time.Millisecond * 3)
			return advanceSlideAnimation{}
		}
	}
	return s, nil
}

func (s slideSwitch) View() string {
	if s.Size.Width == 0 {
		if s.Reverse {
			return s.Left.View()
		}
		return s.Right.View()
	}

	left := strings.Split(s.Left.View(), "\n")
	right := strings.Split(s.Right.View(), "\n")
	boundary := uint(float32(s.Size.Width) * min(s.Progress, 1.0))

	w := strings.Builder{}
	for i := range s.Size.Height {
		if i < len(left) {
			_, _ = w.WriteString(TruncateLeft(left[i], int(boundary), ""))
		} else {
			_, _ = w.WriteString(strings.Repeat(" ", s.Size.Width-int(boundary)-1))
		}
		_, _ = w.WriteString(lipgloss.NormalBorder().Left)
		if i < len(right) {
			_, _ = w.WriteString(truncate.String(right[i], uint(int(boundary)-1)))
		} else {
			_, _ = w.WriteString(strings.Repeat(" ", s.Size.Width-int(boundary)-1))
		}
		if i < s.Size.Height-1 {
			_, _ = w.WriteRune('\n')
		}
	}
	return w.String()
}
