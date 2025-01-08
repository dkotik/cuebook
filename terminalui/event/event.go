package event

import tea "github.com/charmbracelet/bubbletea/v2"

type (
	BackEvent    struct{}
	IsBusyEvent  bool
	SetBusyEvent bool
)

func WithBusySignal(cmd tea.Cmd) tea.Cmd {
	return tea.Sequence(
		func() tea.Msg { return SetBusyEvent(true) },
		cmd,
		func() tea.Msg { return SetBusyEvent(false) },
	)
}

func Propagate(
	msg tea.Msg,
	children []tea.Model,
) (out tea.Cmd) {
	commands := make([]tea.Cmd, 0, len(children))
	for i, child := range children {
		children[i], out = child.Update(msg)
		if out != nil {
			commands = append(commands, out)
		}
	}

	if len(commands) == 0 {
		return nil
	}
	return tea.Batch(commands...)
}

func PropagateInit(
	children []tea.Model,
) (out tea.Cmd) {
	commands := make([]tea.Cmd, 0, len(children))
	for i, child := range children {
		children[i], out = child.Init()
		if out != nil {
			commands = append(commands, out)
		}
	}

	if len(commands) == 0 {
		return nil
	}
	return tea.Batch(commands...)
}
