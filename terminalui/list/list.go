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

func (l List) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Key().Code {
		case tea.KeyEnter:
			return l, func() tea.Msg {
				return SelectionMadeEvent{
					Index: l.SelectedIndex,
				}
			}
		case tea.KeyTab:
			if msg.Key().Mod == tea.ModShift {
				if len(l.Items) > 1 {
					if l.SelectedIndex > 0 {
						return l.applySelection(l.SelectedIndex - 1)
					}
				}
			} else {
				if l.SelectedIndex < len(l.Items)-1 {
					return l.applySelection(l.SelectedIndex + 1)
				}
			}
		case tea.KeyDown, 'j':
			if msg.Key().Mod == tea.ModCtrl {
				maximum := len(l.Items) - 1
				return l, func() tea.Msg {
					if l.SelectedIndex >= maximum {
						return nil
					}
					return SwapOrderEvent{
						CurrentIndex: l.SelectedIndex,
						DesiredIndex: l.SelectedIndex + 1,
					}
				}
			}

			if l.IsFullscreen() && !l.fullScreenView.AtBottom() {
				var cmd tea.Cmd
				l.fullScreenView.HalfViewDown()
				return l, cmd
			}

			if len(l.Items) > 1 {
				if l.SelectedIndex < len(l.Items)-1 {
					return l.applySelection(l.SelectedIndex + 1)
				}
				return l.applySelection(0)
			}
		case tea.KeyUp, 'k':
			if msg.Key().Mod == tea.ModCtrl {
				return l, func() tea.Msg {
					if l.SelectedIndex <= 1 {
						return nil
					}
					return SwapOrderEvent{
						CurrentIndex: l.SelectedIndex,
						DesiredIndex: l.SelectedIndex - 1,
					}
				}
			}

			if l.IsFullscreen() && !l.fullScreenView.AtTop() {
				var cmd tea.Cmd
				*l.fullScreenView, cmd = l.fullScreenView.Update(msg)
				l.fullScreenView.HalfViewUp()
				return l, cmd
			}

			if len(l.Items) > 1 {
				if l.SelectedIndex < 1 {
					return l.applySelection(len(l.Items) - 1)
				}
				return l.applySelection(l.SelectedIndex - 1)
			}
		}
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
		return l.applySelection(l.SelectedIndex)
	case addItemsEvent:
		l.Items = append(l.Items, msg.Items...)
		return l, nil
	case selectedIndexRequestEvent:
		return l, func() tea.Msg {
			return SelectedIndexEvent{
				Index: l.SelectedIndex,
			}
		}
	case applySelectionEvent:
		// && l.SelectedIndex != msg.Index
		if msg.Index >= 0 && msg.Index < len(l.Items) {
			return l.applySelection(msg.Index)
		}
	case tea.WindowSizeMsg:
		l.Size = msg
		msg.Width = msg.Width * 2 / 3
		if msg.Width > 80 {
			msg.Width = 80
		}
		l.fullScreenView.SetWidth(msg.Width)
		l.fullScreenView.SetHeight(msg.Height)
		cmd := event.Propagate(msg, l.Items)
		return l, cmd
	}

	// any other message goes to the selected node
	var cmd tea.Cmd
	if len(l.Items) > 0 {
		l.Items[l.SelectedIndex], cmd = l.Items[l.SelectedIndex].Update(msg)
	}
	return l, cmd
}
