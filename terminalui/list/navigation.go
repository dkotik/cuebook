package list

import tea "github.com/charmbracelet/bubbletea/v2"

func (l List) navigate(msg tea.KeyMsg) (_ tea.Model, cmd tea.Cmd) {
	switch msg.Key().Code {
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

		l, cmd = l.UpdateSelected(msg)
		if cmd == nil {
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
		}
		return l, cmd
	case tea.KeyRight, 'l':
		l, cmd = l.UpdateSelected(msg)
		if cmd == nil {
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
		}
		return l, cmd
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

		l, cmd = l.UpdateSelected(msg)
		if cmd == nil {
			if l.IsFullscreen() && !l.fullScreenView.AtTop() {
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
		return l, cmd
	case tea.KeyLeft, 'h':
		l, cmd = l.UpdateSelected(msg)
		if cmd == nil {
			if l.IsFullscreen() && !l.fullScreenView.AtTop() {
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
		return l, cmd
	}
	return l.UpdateSelected(msg)
}
