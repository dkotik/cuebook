package event

import tea "github.com/charmbracelet/bubbletea/v2"

func If(condition bool, then tea.Cmd) tea.Cmd {
	if condition {
		return then
	}
	return nil
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
