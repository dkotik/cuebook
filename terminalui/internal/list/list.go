package list

import (
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/dkotik/cuebook/terminalui/event"
)

// Configuration controls list content and appearance. Unset Style properties
// inherit the default vertical centering. The list manages its own height and
// left margin to fit the window.
//
// OnSwapOrder converts a Ctrl+J or Ctrl+K reordering request into the message
// the list emits. It defaults to emitting the [SwapOrderEvent] unchanged.
type Configuration struct {
	Items       []tea.Model
	Style       lipgloss.Style
	OnSwapOrder func(SwapOrderEvent) tea.Msg
}

func New(c Configuration) tea.Model {
	if c.OnSwapOrder == nil {
		c.OnSwapOrder = func(e SwapOrderEvent) tea.Msg { return e }
	}
	fs := viewport.New()
	l := List{
		Items:          c.Items,
		Style:          c.Style.Inherit(lipgloss.NewStyle().AlignVertical(lipgloss.Center)),
		fullScreenView: &fs,
		onSwapOrder:    c.OnSwapOrder,
	}
	l, l.initCmd = l.applySelection(l.SelectedIndex)
	return l
}

type List struct {
	Filter        string
	SelectedIndex int
	Items         []tea.Model
	Size          tea.WindowSizeMsg
	Style         lipgloss.Style

	fullScreenView *viewport.Model
	onSwapOrder    func(SwapOrderEvent) tea.Msg
	initCmd        tea.Cmd
}

func (l List) Init() tea.Cmd {
	return tea.Batch(event.PropagateInit(l.Items), l.initCmd)
}

func (l List) IsFullscreen() bool {
	if len(l.Items) == 0 {
		return false
	}
	view := l.Items[l.SelectedIndex].View().Content
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
	case tea.KeyPressMsg:
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
	case addItemsEvent:
		l.Items = append(l.Items, msg...)
		cmd := event.PropagateInit(msg)
		return l, cmd
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
