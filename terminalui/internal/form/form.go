package form

import (
	"github.com/dkotik/cuebook/terminalui/internal/label"
	"github.com/dkotik/cuebook/terminalui/internal/list"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
)

const (
	DefaultLabelMinimumHorizontalWidth  = 20
	DefaultLabelWidthRatioToWindowWidth = 0.15
)

// Configuration controls form layout and appearance. Labels sit beside their
// fields when the label column, sized as a ratio of the window width, is at
// least the minimum width; otherwise they move above their fields. Zero or
// negative values use the defaults. Style and OnSwapOrder are passed to the
// underlying list.
type Configuration struct {
	LabelMinimumHorizontalWidth  int
	LabelWidthRatioToWindowWidth float32
	Style                        lipgloss.Style
	OnSwapOrder                  func(list.SwapOrderEvent) tea.Msg
}

type formLayoutIsHorizontalEvent bool

type SaveChangesEvent struct{}

func New(c Configuration) tea.Model {
	if c.LabelMinimumHorizontalWidth <= 0 {
		c.LabelMinimumHorizontalWidth = DefaultLabelMinimumHorizontalWidth
	}
	if c.LabelWidthRatioToWindowWidth <= 0 {
		c.LabelWidthRatioToWindowWidth = DefaultLabelWidthRatioToWindowWidth
	}
	return form{
		Model: list.New(list.Configuration{
			Style:       c.Style,
			OnSwapOrder: c.OnSwapOrder,
		}),
		LabelMinimumHorizontalWidth:  c.LabelMinimumHorizontalWidth,
		LabelWidthRatioToWindowWidth: c.LabelWidthRatioToWindowWidth,
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
