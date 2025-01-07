package list

import (
	"github.com/charmbracelet/bubbles/v2/viewport"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/dkotik/cuebook/terminalui"
)

func New(name string, items ...tea.Model) tea.Model {
	fs := viewport.New()
	return List{
		Name:  name,
		Items: items,
		// SelectedIndex: -1,
		fullScreenView: &fs,
	}
}

type List struct {
	Name          string
	Filter        string
	SelectedIndex int
	Items         []tea.Model
	Size          tea.WindowSizeMsg

	fullScreenView *viewport.Model
}

func (l List) Init() (m tea.Model, cmd tea.Cmd) {
	m, cmd = l.applySelection(l.SelectedIndex)
	return m, tea.Batch(terminalui.PropagateInit(l.Items), cmd)
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
		// case tea.KeyEnter:
		// 	event := applySelectionEvent{Index: l.SelectedIndex}
		// 	return l, func() tea.Msg {
		// 		return event
		// 	}
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
			if l.IsFullscreen() && !l.fullScreenView.AtBottom() {
				var cmd tea.Cmd
				l.fullScreenView.HalfViewDown()
				return l, cmd
			}

			if len(l.Items) > 1 {
				if l.SelectedIndex < len(l.Items)-1 {
					return l.applySelection(l.SelectedIndex + 1)
				}
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
					return l.applySelection(l.SelectedIndex - 1)
				}
			}
		}
	case resetEvent:
		if msg.ListName == l.Name {
			l.SelectedIndex = 0
			l.Items = nil
			return l, nil
		}
	case addItemsEvent:
		if msg.ListName == l.Name {
			l.Items = append(l.Items, msg.Items...)
			return l, nil
		}
	case applySelectionEvent:
		// && l.SelectedIndex != msg.Index
		if msg.ListName == l.Name && msg.Index >= 0 && msg.Index < len(l.Items) {
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
		cmd := terminalui.Propagate(msg, l.Items)
		return l, cmd
	}

	// any other message goes to the selected node
	var cmd tea.Cmd
	if len(l.Items) > 0 {
		l.Items[l.SelectedIndex], cmd = l.Items[l.SelectedIndex].Update(msg)
	}
	return l, cmd
}
