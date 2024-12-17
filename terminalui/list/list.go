package list

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/dkotik/cuebook/terminalui/card"
)

func New(cards ...card.Card) tea.Model {
	return list{
		Items: cards,
	}
}

type list struct {
	Filter        string
	SelectedIndex int
	Items         []card.Card
}

func (l list) Init() (tea.Model, tea.Cmd) {
	return l, nil
}

func (l list) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Key().Code {
		case tea.KeyDown, 'j':
			if len(l.Items) > 1 {
				if l.SelectedIndex < len(l.Items)-1 {
					l.Items[l.SelectedIndex].Selected = false
					l.SelectedIndex++
					l.Items[l.SelectedIndex].Selected = true
				}
				return l, nil
			}
		case tea.KeyUp, 'k':
			if len(l.Items) > 1 {
				if l.SelectedIndex > 0 {
					l.Items[l.SelectedIndex].Selected = false
					l.SelectedIndex--
					l.Items[l.SelectedIndex].Selected = true
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
			l.Items[i] = model.(card.Card)
			cmds = append(cmds, cmd)
		}

		return l, tea.Batch(cmds...)
	}
	return l, nil
}

func (l list) View() string {
	total := len(l.Items)
	if total == 0 {
		return "</>"
	}
	if l.SelectedIndex >= total {
		l.SelectedIndex = total - 1
	}
	return l.Items[l.SelectedIndex].View()
}
