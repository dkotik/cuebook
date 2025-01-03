package status

import (
	"github.com/charmbracelet/bubbles/v2/help"
	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/spinner"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/dkotik/cuebook/terminalui"
)

func New(bindings ...key.Binding) tea.Model {
	return model{
		KeyBindings: bindings,
		Spinner: spinner.New(
			spinner.WithSpinner(spinner.MiniDot),
			spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("4"))),
		),
		Help: help.New(),
	}
}

type model struct {
	KeyBindings []key.Binding
	Spinner     spinner.Model
	Help        help.Model
	Busy        bool
	Success     bool
	Error       error
}

func (m model) Init() (tea.Model, tea.Cmd) {
	return m, tea.Batch(
		m.Spinner.Tick,
		func() tea.Msg {
			return terminalui.IsBusyEvent(true)
		},
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case terminalui.IsBusyEvent:
		m.Busy = bool(msg)
		if m.Busy {
			return m, m.Spinner.Tick
		}
	case spinner.TickMsg:
		// if m.Busy {
		var cmd tea.Cmd
		m.Spinner, cmd = m.Spinner.Update(msg)
		return m, cmd
		// }
	}
	return m, nil
}
