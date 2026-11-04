package form

import (
	"github.com/dkotik/cuebook/terminalui/internal/label"
	"github.com/dkotik/cuebook/terminalui/list"

	tea "github.com/charmbracelet/bubbletea/v2"
)

type formLayoutIsHorizontalEvent bool

type SaveChangesEvent struct{}

func New() tea.Model {
	// fields ...tea.Model
	// layout := list.New()
	// layout, cmd := layout.Update(list.AddItems(fields...))

	return form{
		Model:                        list.New(),
		LabelMinimumHorizontalWidth:  20,
		LabelWidthRatioToWindowWidth: 0.15,
	}
}

type form struct {
	tea.Model
	WindowSize                   tea.WindowSizeMsg
	LabelMinimumHorizontalWidth  int
	LabelWidthRatioToWindowWidth float32
}

func (f form) Init() (_ tea.Model, cmd tea.Cmd) {
	f.Model, cmd = f.Model.Init()
	return f, tea.Batch(cmd, tea.RequestWindowSize())
}

func (f form) Update(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Key().Code {
		case tea.KeyEscape:
			return f, nil
		case 's':
			if msg.Key().Mod.Contains(tea.ModCtrl) {
				// f.Model, cmd = f.Model.Update(msg)
				// if cmd == nil {
				// }
				// return f, cmd
				return f, func() tea.Msg { return SaveChangesEvent{} }
			}
		}
	case tea.WindowSizeMsg:
		if msg != f.WindowSize {
			f.WindowSize = msg
			labelWidth := int(float32(f.WindowSize.Width) * f.LabelWidthRatioToWindowWidth)
			f.Model, _ = f.Model.Update(label.LayoutChangeEvent{
				Width:        labelWidth,
				IsHorizontal: labelWidth >= f.LabelMinimumHorizontalWidth,
			})
		}
	}
	f.Model, cmd = f.Model.Update(msg)
	return f, cmd
}
