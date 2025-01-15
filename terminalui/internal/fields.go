package internal

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/patch"
	"github.com/dkotik/cuebook/terminalui/field"
	"github.com/dkotik/cuebook/terminalui/list"
)

type (
	fieldHighlighted int
	fieldListCards   []tea.Model
)

type FieldList struct {
	tea.Model

	book patch.Result
	// selected int // *patch.ByteAnchor
}

func (l FieldList) Init() (_ tea.Model, cmd tea.Cmd) {
	l.Model = list.New()
	// l.selected = -1
	return l, nil
}

func (l FieldList) Update(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	switch msg := msg.(type) {
	case fieldHighlighted:
		// l.selected = int(msg)
		return l, nil
	case cuebook.Entry:
		return l, LoadFields(msg) // TODO: track patch changes
	case fieldListCards:
		l.Model, cmd = l.Model.Init()
		var setCmd tea.Cmd
		l.Model, setCmd = l.Model.Update(list.SetItems(msg...)())
		return l, tea.Sequence(cmd, setCmd, tea.RequestWindowSize())
	// case Book:
	// 	return l, func() tea.Msg {
	// 		// use func (b Book) LastDifferentEntry() ?
	// 	}
	case tea.KeyMsg:
		l.Model, cmd = l.Model.Update(msg)
		if msg.Key().Code == tea.KeyEnter {
			return l, nil // TODO: open form
		}
		return l, NewListItemHighlightAdaptor[fieldHighlighted](cmd)
	default:
		l.Model, cmd = l.Model.Update(msg)
		return l, cmd
	}
}

const (
	fieldEditingTextAreaName = "fieldEditingTextArea"
	fieldAddingTextAreaName  = "fieldAddingTextArea"
)

func LoadFields(entry cuebook.Entry) tea.Cmd {
	return func() tea.Msg {
		// entry, err := cuebook.NewEntry(book.Value.LookupPath(cue.MakePath(cue.Index(index))))
		// if err != nil {
		// 	return err
		// }
		// total, err := book.Len()
		// if err != nil {
		// 	return err
		// }
		fields := make([]tea.Model, 0, len(entry.Fields)+len(entry.Details)+1)
		// fields = append(fields, list.Title{
		// 	Text:  entry.GetTitle() + fmt.Sprintf(" › %d/%d", index+1, total),
		// 	Style: lipgloss.NewStyle().Bold(true).Align(lipgloss.Left).Foreground(lipgloss.BrightRed),
		// })
		for _, f := range entry.Fields {
			fields = append(fields, field.New(f.Name, f.String()))
		}
		for _, f := range entry.Details {
			fields = append(fields, field.New(f.Name, f.String()))
		}
		return fieldListCards(fields)
		// return tea.BatchMsg{
		// tea.Sequence(
		// 	list.Reset(entryFieldListName),
		// 	list.AddItems(entryFieldListName, fields...),
		// 	tea.RequestWindowSize(),
		// 	list.ApplySelection(entryFieldListName, index+1), // +1 for title
		// ),
		// }
	}
}
