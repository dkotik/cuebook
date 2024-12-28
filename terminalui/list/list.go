package list

import (
	"github.com/charmbracelet/bubbles/v2/viewport"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/dkotik/cuebook/terminalui"
)

func New(items ...tea.Model) tea.Model {
	fs := viewport.New()
	return List{
		Items: items,
		// SelectedIndex: -1,
		fullScreenView: &fs,
	}
}

type List struct {
	Filter        string
	SelectedIndex int
	Items         []tea.Model
	Size          tea.WindowSizeMsg

	fullScreenView *viewport.Model
}

func (l List) Init() (tea.Model, tea.Cmd) {
	return l, terminalui.PropagateInit(l.Items)
}

func (l List) IsFullscreen() bool {
	view := l.Items[l.SelectedIndex].View()
	// l.fullScreenView.SetContent(view)
	return lipgloss.Height(view) > l.Size.Height
}

func (l List) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Key().Code {
		case tea.KeyEnter:
			event := selectEvent{Index: l.SelectedIndex}
			return l, func() tea.Msg {
				return event
			}
		case tea.KeyDown, 'j':
			if l.IsFullscreen() && !l.fullScreenView.AtBottom() {
				var cmd tea.Cmd
				l.fullScreenView.HalfViewDown()
				return l, cmd
			}

			if len(l.Items) > 1 {
				if l.SelectedIndex < len(l.Items)-1 {
					l.SelectedIndex++
				}
				return l, nil
			}
		case tea.KeyUp, 'k':
			if l.IsFullscreen() && !l.fullScreenView.AtTop() {
				var cmd tea.Cmd
				*l.fullScreenView, cmd = l.fullScreenView.Update(msg)
				l.fullScreenView.HalfViewUp()
				return l, cmd
			}

			if len(l.Items) > 1 {
				if l.SelectedIndex > 0 {
					l.SelectedIndex--
				}
				return l, nil
			}
		}
	case tea.WindowSizeMsg:
		l.Size = msg
		if msg.Width > 80 {
			msg.Width = 80
		}
		l.fullScreenView.SetWidth(msg.Width)
		l.fullScreenView.SetHeight(msg.Height)
		cmd := terminalui.Propagate(msg, l.Items)
		return l, cmd
	}
	return l, nil
}
