package terminalui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea/v2"
)

type testParentModel struct {
	first    tea.Model
	second   tea.Model
	children []tea.Model
}

type testChildModel struct {
	updated bool
}

func TestPropagateEvent(t *testing.T) {
	parent := testParentModel{
		first:  testChildModel{},
		second: testChildModel{},
		children: []tea.Model{
			testChildModel{},
			testChildModel{},
		},
	}

	Propagate(func() tea.Msg {
		return true
	}, parent.children)

	for _, child := range parent.children {
		if !child.(testChildModel).updated {
			t.Error("child model not updated")
		}
	}

	Propagate(func() tea.Msg {
		return true
	}, []tea.Model{parent.first, parent.second})
	if parent.first.(testChildModel).updated {
		t.Error("child model was updated, when it should not have been")
	}
	if parent.second.(testChildModel).updated {
		t.Error("child model was updated, when it should not have been")
	}
}

func (c testChildModel) Init() (tea.Model, tea.Cmd) {
	return c, nil
}

func (c testChildModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	c.updated = true
	return c, nil
}

func (c testChildModel) View() string {
	return ""
}
