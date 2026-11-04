package textarea

import (
	"github.com/charmbracelet/bubbles/v2/textarea"
	tea "github.com/charmbracelet/bubbletea/v2"
)

func ScrollFix(ta textarea.Model) textarea.Model {
	// scrolling up, then down to match viewpower to height
	// because f.Input.Cursor only has line jumps to front or end
	// TODO: contribute cursor start and end of input jumps
	// see: https://github.com/charmbracelet/bubbles/pull/559
	ta, _ = ta.Update(tea.KeyPressMsg{Code: tea.KeyHome, Mod: tea.ModCtrl, Text: "ctrl+home"})
	ta, _ = ta.Update(tea.KeyPressMsg{Code: tea.KeyEnd, Mod: tea.ModCtrl, Text: "ctrl+end"})
	return ta
}
