package entry

import (
	"fmt"

	"cuelang.org/go/cue"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/dkotik/cuebook"
	listForm "github.com/dkotik/cuebook/terminalui/form"
	"github.com/dkotik/cuebook/terminalui/list"
	"github.com/dkotik/cuebook/terminalui/window"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

type createForm struct {
	tea.Model
}

type createFormFields []tea.Model

func emitCreateFormFields(value cue.Value) tea.Cmd {
	return func() tea.Msg {
		fields := make(createFormFields, 9)
		for selector, field := range cuebook.EachFieldDefinition(value) {
			name := selector.Unquoted()
			fields = append(fields, listForm.NewField(name, field.Kind().String(), func(updated string) tea.Cmd {
				original := "f.String()"
				return tea.Batch(
					func() tea.Msg {
						return fieldChangedEvent{
							Name:     name,
							Value:    updated,
							Original: original,
						}
					},
				)
			}))
		}
		return fields
	}
}

func NewCreateForm(value cue.Value) tea.Cmd {
	return tea.Sequence(
		func() tea.Msg {
			return window.SwitchTo(createForm{})
		},
		emitCreateFormFields(value),
	)
}

func (f createForm) Init() (_ tea.Model, cmd tea.Cmd) {
	f.Model, cmd = listForm.New().Init()
	return f, cmd
}

func (f createForm) Update(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	switch msg := msg.(type) {
	case createFormFields:
		f.Model, cmd = f.Model.Update(list.SetItems(msg...)())
		return f, tea.Sequence(cmd, tea.RequestWindowSize(), window.RequestLocalizer())
	case listForm.SaveChangesEvent:
		fmt.Sprint(msg)
		return f, func() tea.Msg { return window.BackEvent{} }
	}
	f.Model, cmd = f.Model.Update(msg)
	return f, cmd
}

// func (f createForm) View() string {
// 	return fmt.Sprintf("%d", 45444444444)
// }

var createButtonText = &i18n.LocalizeConfig{
	DefaultMessage: &i18n.Message{
		ID:    "bookEntryCreate",
		Other: "Create a New Entry",
	},
}

type CreateEvent struct{}

func NewCreateButton() tea.Model {
	return listForm.NewBlankResponsiveLabel(window.NewTranslatableModel(createButton{}))
}

type createButton struct {
	tea.Model
}

func (d createButton) Init() (tea.Model, tea.Cmd) {
	return d, nil
}

func (d createButton) Translate(lc *i18n.Localizer) (window.TranslatableModel, error) {
	text, err := lc.Localize(createButtonText)
	if err != nil {
		return nil, err
	}
	d.Model = list.NewButton(text, func() tea.Msg { return CreateEvent{} })
	return d, nil
}

func (d createButton) Update(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	if d.Model == nil {
		return d, nil
	}
	if _, ok := msg.(fieldChangedEvent); ok {
		d.Model, cmd = cancelButton{}.Init()
		return d.Model, tea.Batch(cmd, tea.RequestWindowSize(), window.RequestLocalizer())
	}
	d.Model, cmd = d.Model.Update(msg)
	return d, cmd
}

func (d createButton) View() string {
	if d.Model == nil {
		return "..."
	}
	return d.Model.View()
}
