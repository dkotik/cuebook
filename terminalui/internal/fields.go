package internal

import (
	"maps"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/patch"
	"github.com/dkotik/cuebook/terminalui/form"
	"github.com/dkotik/cuebook/terminalui/list"
	"github.com/dkotik/cuebook/terminalui/textarea"
	"github.com/dkotik/cuebook/terminalui/window"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

type (
	fieldHighlighted int
	fieldListCards   []tea.Model

	fieldChangedEvent struct {
		Name     string
		Value    string
		Original string
	}

	applyFieldChanges struct{}

	updateFieldPatch struct {
		patch.Patch
		Entry cuebook.Entry
	}

	swapEntriesPatch struct {
		patch.Patch
		// Target patch.ByteAnchor
	}

	swapFieldsPatch struct {
		patch.Patch
	}

	deleteEntryPatch struct {
		patch.Patch
		Entry cuebook.Entry
	}
)

type FieldList struct {
	tea.Model

	changes  map[string]string
	state    patch.Result
	entry    cuebook.Entry
	selected int
}

func (l FieldList) Init() (_ tea.Model, cmd tea.Cmd) {
	l.Model, cmd = form.New().Init()
	l.changes = make(map[string]string)
	return l, cmd
}

func (l FieldList) Update(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	switch msg := msg.(type) {
	case fieldChangedEvent:
		if msg.Value == msg.Original {
			delete(l.changes, msg.Name)
		} else {
			l.changes[msg.Name] = msg.Value
		}
		return l, nil
	case applyFieldChanges:
		changes := maps.Clone(l.changes)
		return l, func() tea.Msg {
			p, err := patch.MergeFieldValues(l.state.Source, l.entry.Value, changes)
			if err != nil {
				return err
			}
			return updateFieldPatch{
				Patch: p,
				Entry: l.entry,
			}
		}
	case fieldHighlighted:
		l.selected = int(msg)
		return l, nil
	case patch.Result:
		// if !l.state.IsEqual(msg) { }
		l.state = msg
		if _, ok := msg.LastChange.(updateFieldPatch); ok {
			return l, func() tea.Msg { return window.BackEvent{} }
		}
		if _, ok := msg.LastChange.(deleteEntryPatch); ok {
			return l, func() tea.Msg { return window.BackEvent{} }
		}
		return l, nil
	case cuebook.Entry:
		l.entry = msg
		return l, LoadFields(l.state.Source, msg)
	case fieldListCards:
		l.Model, cmd = form.New().Init()
		var setCmd tea.Cmd
		l.Model, setCmd = l.Model.Update(list.SetItems(msg...)())
		return l, tea.Sequence(cmd, setCmd)
	case list.SwapOrderEvent:
		return l, func() tea.Msg {
			a, err := l.entry.GetField(msg.CurrentIndex)
			if err != nil {
				return err
			}
			b, err := l.entry.GetField(msg.DesiredIndex)
			if err != nil {
				return err
			}
			p, err := patch.SwapEntries(l.state.Source, a.Value, b.Value)
			if err != nil {
				return err
			}
			return swapFieldsPatch{Patch: p}
		}
	default:
		l.Model, cmd = l.Model.Update(msg)
		return l, cmd
	}
}

func displayFieldForm(source []byte, entry cuebook.Entry, field cuebook.Field) tea.Cmd {
	return func() tea.Msg {
		form, err := textarea.New(
			textarea.WithLabel(field.Name),
			textarea.WithValue(field.String()),
			textarea.WithOnSubmitCommand(func(value string) tea.Cmd {
				return func() tea.Msg {
					p, err := patch.UpdateFieldValue(source, entry.Value, field.Value, value)
					if err != nil {
						return err
					}
					return updateFieldPatch{
						Patch: p,
						Entry: entry,
					}
				}
			}),
		)
		if err != nil {
			return err
		}
		return window.SwitchTo(form)
	}
}

func createField(f cuebook.Field) tea.Model {
	return form.NewField(f.Name, f.String(), func(updated string) tea.Cmd {
		original := f.String()
		return func() tea.Msg {
			return fieldChangedEvent{
				Name:     f.Name,
				Value:    updated,
				Original: original,
			}
		}
	})
}

func LoadFields(source []byte, entry cuebook.Entry) tea.Cmd {
	return func() tea.Msg {
		return window.TranslatableFunc(func(lc *i18n.Localizer) tea.Cmd {
			fields := make([]tea.Model, 0, len(entry.Fields)+len(entry.Details)+1)
			// fields = append(fields, list.Title{
			// 	Text:  entry.GetTitle() + fmt.Sprintf(" › %d/%d", index+1, total),
			// 	Style: lipgloss.NewStyle().Bold(true).Align(lipgloss.Left).Foreground(lipgloss.BrightRed),
			// })
			for _, f := range entry.Fields {
				fields = append(fields, createField(f))
			}
			for _, f := range entry.Details {
				fields = append(fields, createField(f))
			}
			commitLabel := lc.MustLocalize(&i18n.LocalizeConfig{
				DefaultMessage: &i18n.Message{
					ID:    "bookEntryCommitChanges",
					Other: "Save Changes",
				},
			})
			rmLabel := lc.MustLocalize(&i18n.LocalizeConfig{
				DefaultMessage: &i18n.Message{
					ID:    "bookEntryDelete",
					Other: "Delete Entry",
				},
			})
			return func() tea.Msg {
				return fieldListCards(append(
					fields,
					// TODO: share label with all buttons
					form.NewBlankResponsiveLabel(list.NewButton(commitLabel, func() tea.Msg {
						return applyFieldChanges{}
					})),
					form.NewBlankResponsiveLabel(list.NewButton(rmLabel, func() tea.Msg {
						p, err := patch.DeleteFromStructList(source, entry.Value)
						if err != nil {
							return err
						}
						return deleteEntryPatch{
							Patch: p,
							Entry: entry,
						}
					})),
				))
			}
		})
	}
}
