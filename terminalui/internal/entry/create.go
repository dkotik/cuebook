package entry

import (
	tea "charm.land/bubbletea/v2"
	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/cuecontext"
	"github.com/dkotik/cuebook"
	listForm "github.com/dkotik/cuebook/terminalui/internal/form"
	"github.com/dkotik/cuebook/terminalui/internal/label"
	"github.com/dkotik/cuebook/terminalui/internal/list"
	"github.com/dkotik/cuebook/terminalui/window"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

type createEntryPatch struct {
	cuebook.Patch
}

type createForm struct {
	tea.Model
	Source []byte
}

type createFormFields []tea.Model

func emitCreateFormFields(value cue.Value) tea.Cmd {
	return func() tea.Msg {
		fields := make(createFormFields, 0, 9)
		for definition := range cuebook.EachFieldDefinition(value) {
			name := definition.Selector.Unquoted()
			text, _ := definition.Value.String()
			if text == "" {
				def, _ := value.Default()
				text, _ = def.String()
				// TODO: apply @cuebook default
			}

			fields = append(fields,
				listForm.NewField(name, text, func(updated string) tea.Cmd {
					original := text
					return tea.Batch(
						func() tea.Msg {
							return fieldChangedEvent{
								Name:     name,
								Value:    updated,
								Original: original,
							}
						},
					)
				}),
			)
		}
		return fields
	}
}

func NewCreateForm(source []byte, value cue.Value) tea.Cmd {
	return tea.Sequence(
		func() tea.Msg {
			return window.SwitchTo(newCreateForm(source))
		},
		emitCreateFormFields(value),
	)
}

func newCreateForm(source []byte) createForm {
	return createForm{Model: listForm.New(listForm.Configuration{}), Source: source}
}

func (f createForm) Init() tea.Cmd {
	return f.Model.Init()
}

func (f createForm) Update(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	switch msg := msg.(type) {
	case createFormFields:
		f.Model, cmd = f.Model.Update(list.SetItems(msg...)())
		return f, tea.Sequence(cmd, window.RequestWindowSize(), window.RequestLocalizer())
	case cuebook.Result:
		f.Source = msg.Source
		if _, ok := msg.LastChange.(createEntryPatch); ok {
			return f, func() tea.Msg { return window.BackEvent{} }
		}
	case listForm.SaveChangesEvent:
		return f, func() tea.Msg {
			p, err := cuebook.AppendToStructList(f.Source, cuecontext.New().BuildExpr(
				ast.NewStruct(
					&ast.Field{
						Label: ast.NewString("Name"),
						Value: ast.NewString("Someone!!!!"),
					},
					&ast.Field{
						Label: ast.NewString("Email"),
						Value: ast.NewString("someEmail@somehost.net"),
					},
				),
			))
			if err != nil {
				return err
			}
			return createEntryPatch{p}
		}
	}
	f.Model, cmd = f.Model.Update(msg)
	return f, cmd
}

var createButtonText = &i18n.LocalizeConfig{
	DefaultMessage: &i18n.Message{
		ID:    "bookEntryCreate",
		Other: "Create a New Entry",
	},
}

type CreateEvent struct{}

func NewCreateButton() tea.Model {
	return label.New(window.NewTranslatableModel(createButton{}), nil)
}

type createButton struct {
	tea.Model
}

func (d createButton) Init() tea.Cmd {
	return nil
}

func (d createButton) Translate(lc *i18n.Localizer) (window.TranslatableModel, error) {
	d.Model = list.NewButton(createButtonText, func() tea.Msg { return CreateEvent{} })
	return d, nil
}

func (d createButton) Update(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	if d.Model == nil {
		return d, nil
	}
	d.Model, cmd = d.Model.Update(msg)
	return d, cmd
}

func (d createButton) View() tea.View {
	if d.Model == nil {
		return tea.NewView("...")
	}
	return d.Model.View()
}
