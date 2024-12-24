package list

func (l list) View() string {
	total := len(l.Items)
	if total == 0 {
		return "</>"
	}
	return l.Items[l.SelectedIndex].View()
}
