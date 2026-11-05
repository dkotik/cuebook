package status

import (
	"fmt"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/dkotik/cuebook/terminalui/window"
)

// Configuration controls the status bar content and appearance. Unset style
// properties inherit the defaults: a blue spinner and the dark help styles.
type Configuration struct {
	KeyBindings []key.Binding

	SpinnerStyle     lipgloss.Style
	KeyStyle         lipgloss.Style
	DescriptionStyle lipgloss.Style
	SeparatorStyle   lipgloss.Style
	EllipsisStyle    lipgloss.Style
}

func New(c Configuration) tea.Model {
	h := help.New()
	d := h.Styles
	h.Styles.ShortKey = c.KeyStyle.Inherit(d.ShortKey)
	h.Styles.ShortDesc = c.DescriptionStyle.Inherit(d.ShortDesc)
	h.Styles.ShortSeparator = c.SeparatorStyle.Inherit(d.ShortSeparator)
	h.Styles.Ellipsis = c.EllipsisStyle.Inherit(d.Ellipsis)
	return model{
		KeyBindings: c.KeyBindings,
		Spinner: spinner.New(
			spinner.WithSpinner(spinner.MiniDot),
			spinner.WithStyle(c.SpinnerStyle.Inherit(lipgloss.NewStyle().Foreground(lipgloss.Color("4")))),
		),
		Help: h,
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

func (m model) Init() tea.Cmd {
	return nil // m.Spinner.Tick
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case error:
		panic(fmt.Errorf("caught error: %w", msg))
	case window.IsBusyEvent:
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
