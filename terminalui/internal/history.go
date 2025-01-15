package internal

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/dkotik/cuebook/patch"
)

type historyPatch struct {
	patch.Patch
}

type snapShot struct {
	Patch              patch.Patch
	SelectedEntryIndex int
	SelectedFieldIndex int
}

type patchHistoryTracker struct {
	history    []snapShot
	cursor     int
	entryIndex int
	fieldIndex int
}

func (h patchHistoryTracker) undo() (tea.Model, tea.Cmd) {
	if len(h.history) == 0 || h.cursor < 0 {
		return h, nil
	}
	patch := h.history[h.cursor]
	h.cursor--
	return h, tea.Sequence(
		func() tea.Msg { return historyPatch{patch.Patch.Invert()} },
		// list.ApplySelection(entryFieldListName, h.fieldIndex),
		// list.ApplySelection(entryListName, h.entryIndex),
	)
}

func (h patchHistoryTracker) redo() (tea.Model, tea.Cmd) {
	total := len(h.history)
	if total == 0 || h.cursor >= total {
		return h, nil
	}
	patch := h.history[h.cursor]
	h.cursor++
	return h, tea.Sequence(
		func() tea.Msg { return historyPatch{patch.Patch} },
		// list.ApplySelection(entryFieldListName, h.fieldIndex),
		// list.ApplySelection(entryListName, h.entryIndex),
	)
}

func (h patchHistoryTracker) Init() (tea.Model, tea.Cmd) {
	return h, nil
}

func (h patchHistoryTracker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Key().Code == 'z' {
			if msg.Key().Mod == tea.ModShift {
				return h.redo()
			}
			return h.undo()
		}
	case historyPatch:
		// ignore all patches issued by this component
	case patch.Patch:
		total := len(h.history)
		if total > 5 {
			h.history = h.history[1 : h.cursor+1]
		} else if total > 0 {
			h.history = h.history[:h.cursor+1]
		}
		h.cursor = len(h.history) - 1
		h.history = append(h.history, snapShot{
			Patch:              msg,
			SelectedEntryIndex: h.entryIndex,
			SelectedFieldIndex: h.fieldIndex,
		})
	}
	return h, nil
}

func (h patchHistoryTracker) View() string { return "<patch history>" }
