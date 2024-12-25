package list

import (
	tea "github.com/charmbracelet/bubbletea/v2"
)

func New(items ...tea.Model) tea.Model {
	return list{
		Items: items,
	}
}

type list struct {
	Filter        string
	SelectedIndex int
	Items         []tea.Model
}

func (l list) Init() (tea.Model, tea.Cmd) {
	return l, nil
}

func (l list) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Key().Code {
		case tea.KeyEnter:
			event := SelectEvent{Index: l.SelectedIndex}
			return l, func() tea.Msg {
				return event
			}
		case tea.KeyDown, 'j':
			if len(l.Items) > 1 {
				if l.SelectedIndex < len(l.Items)-1 {
					l.SelectedIndex++
				}
				return l, nil
			}
		case tea.KeyUp, 'k':
			if len(l.Items) > 1 {
				if l.SelectedIndex > 0 {
					l.SelectedIndex--
				}
				return l, nil
			}
		}
	case tea.WindowSizeMsg:
		var cmds []tea.Cmd
		var cmd tea.Cmd
		var model tea.Model
		for i, item := range l.Items {
			model, cmd = item.Update(msg)
			l.Items[i] = model
			cmds = append(cmds, cmd)
		}

		return l, tea.Batch(cmds...)
	}
	return l, nil
}
