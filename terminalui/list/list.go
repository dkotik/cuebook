package list

import (
	"github.com/charmbracelet/bubbles/v2/viewport"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/dkotik/cuebook/terminalui/event"
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

func (l List) Init() (m tea.Model, cmd tea.Cmd) {
	m, cmd = l.applySelection(l.SelectedIndex) // TODO: deprecate?
	return m, tea.Batch(event.PropagateInit(l.Items), cmd)
}

func (l List) IsFullscreen() bool {
	if len(l.Items) == 0 {
		return false
	}
	view := l.Items[l.SelectedIndex].View()
	// l.fullScreenView.SetContent(view)
	return lipgloss.Height(view) > l.Size.Height
}

func (l List) UpdateSelected(msg tea.Msg) (_ List, cmd tea.Cmd) {
	if len(l.Items) > 0 {
		l.Items[l.SelectedIndex], cmd = l.Items[l.SelectedIndex].Update(msg)
	}
	return l, cmd
}

func (l List) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return l.navigate(msg)
	case countRequestEvent:
		return l, func() tea.Msg {
			count := len(l.Items)
			return CountEvent{
				Count: count,
			}
		}
	case resetEvent:
		l.SelectedIndex = 0
		l.Items = nil
		return l, nil
	case setItemsEvent:
		l.Items = msg
		if l.SelectedIndex >= len(msg) {
			l.SelectedIndex = len(msg) - 1
			// return l.applySelection(l.SelectedIndex)
		}
		// initCmd := event.PropagateInit(l.Items)
		return l.applySelection(l.SelectedIndex)
		// return m, tea.Sequence(
		// 	initCmd,
		// 	cmd,
		// )
	// case addItemsEvent:
	// 	l.Items = append(l.Items, msg...)
	// 	initCmd := event.PropagateInit(l.Items)
	// 	m, cmd := l.applySelection(l.SelectedIndex)
	// 	return m, tea.Sequence(
	// 		initCmd,
	// 		cmd,
	// 	)
	case applySelectionEvent:
		// && l.SelectedIndex != msg.Index
		index := int(msg)
		if index >= 0 && index < len(l.Items) {
			return l.applySelection(index)
		}
	case tea.WindowSizeMsg:
		l.Size = msg
		msg.Width = msg.Width * 2 / 3
		if msg.Width > 80 {
			msg.Width = 80
		}
		l.fullScreenView.SetWidth(msg.Width)
		l.fullScreenView.SetHeight(msg.Height)
		return l, event.Propagate(msg, l.Items)
	}

	// any other message goes to the selected node
	// var cmd tea.Cmd
	// if len(l.Items) > 0 {
	// 	l.Items[l.SelectedIndex], cmd = l.Items[l.SelectedIndex].Update(msg)
	// }
	return l, event.Propagate(msg, l.Items)
}
