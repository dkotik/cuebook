package terminalui

import tea "github.com/charmbracelet/bubbletea/v2"

type (
	ErrorEvent   error
	BackEvent    struct{}
	IsBusyEvent  bool
	setBusyEvent bool
)

func WithBusySignal(cmd tea.Cmd) tea.Cmd {
	return tea.Sequence(
		func() tea.Msg { return setBusyEvent(true) },
		cmd,
		func() tea.Msg { return setBusyEvent(false) },
	)
}

func Propagate(
	cmd tea.Msg,
	children ...*tea.Model,
) (out tea.Cmd) {
	commands := make([]tea.Cmd, 0, len(children))
	for i, child := range children {
		*children[i], out = (*child).Update(cmd)
		if out != nil {
			commands = append(commands, out)
		}
	}

	if len(commands) == 0 {
		return nil
	}
	return tea.Batch(commands...)
}
