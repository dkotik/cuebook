package entry

import (
	"errors"
	"fmt"

	tea "charm.land/bubbletea/v2"
	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/cuecontext"
	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/metadata"
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
	Source      []byte
	Definitions []cuebook.FieldDefinition
	Values      map[string]string
}

type createFormFields struct {
	Items       []tea.Model
	Definitions []cuebook.FieldDefinition
	Values      map[string]string
}

func emitCreateFormFields(value cue.Value) tea.Cmd {
	return func() tea.Msg {
		fields := createFormFields{
			Items:       make([]tea.Model, 0, 9),
			Definitions: make([]cuebook.FieldDefinition, 0, 9),
			Values:      make(map[string]string),
		}
		for definition := range cuebook.EachFieldDefinition(value) {
			name := definition.Selector.Unquoted()
			field := cuebook.NewField(name, definition.Value)
			optional := definition.Selector.ConstraintType() == cue.OptionalConstraint
			text := createFieldInitialValue(field, optional)
			fields.Definitions = append(fields.Definitions, definition)
			fields.Values[name] = text
			fields.Items = append(fields.Items,
				listForm.NewField(name, text, func(updated string) tea.Cmd {
					original := text
					return tea.Batch(func() tea.Msg {
						return fieldChangedEvent{
							Name:     name,
							Value:    updated,
							Original: original,
						}
					})
				}),
			)
		}
		return fields
	}
}

func createFieldInitialValue(field cuebook.Field, optional bool) string {
	if value, ok := field.Default(); ok {
		return value
	}
	if optional {
		return ""
	}
	switch kind := field.Value.IncompleteKind(); {
	case kind&cue.NumberKind != 0:
		return "0"
	case kind&cue.BoolKind != 0:
		return "false"
	case kind&cue.ListKind != 0:
		return "[]"
	case kind&cue.StructKind != 0:
		return "{}"
	default:
		return ""
	}
}

func createFormItems(fields []tea.Model) []tea.Model {
	items := append([]tea.Model(nil), fields...)
	return append(items, NewCreateEntryButton(), NewCancelButton())
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
	case tea.KeyPressMsg:
		if msg.Key().Code == tea.KeyEscape {
			// A nil command makes the window treat Escape as a request to go back.
			return f, func() tea.Msg { return nil }
		}
	case createFormFields:
		f.Definitions = msg.Definitions
		f.Values = msg.Values
		f.Model, cmd = f.Model.Update(list.SetItems(createFormItems(msg.Items)...)())
		return f, tea.Sequence(cmd, window.RequestWindowSize(), window.RequestLocalizer())
	case fieldChangedEvent:
		if f.Values == nil {
			f.Values = make(map[string]string)
		}
		f.Values[msg.Name] = msg.Value
	case cuebook.Result:
		f.Source = msg.Source
		if _, ok := msg.LastChange.(createEntryPatch); ok {
			return f, func() tea.Msg { return window.BackEvent{} }
		}
	case listForm.SaveChangesEvent:
		return f, func() tea.Msg {
			declarations := make([]any, 0, len(f.Definitions))
			for _, definition := range f.Definitions {
				name := definition.Selector.Unquoted()
				field := cuebook.NewField(name, definition.Value)
				optional := definition.Selector.ConstraintType() == cue.OptionalConstraint
				value, exists := f.Values[name]
				if !exists {
					if optional {
						continue
					}
					return fmt.Errorf("required field %q has no value", name)
				}
				if optional && value == "" {
					continue
				}
				declaration, err := createEntryFieldDeclaration(field, value)
				if err != nil {
					return fmt.Errorf("unable to format field %q: %w", name, err)
				}
				declarations = append(declarations, declaration)
			}

			entry := cuecontext.New().BuildExpr(ast.NewStruct(declarations...))
			if err := entry.Err(); err != nil {
				return fmt.Errorf("unable to build entry: %w", err)
			}
			p, err := cuebook.AppendToStructList(f.Source, entry)
			if err != nil {
				return err
			}
			return createEntryPatch{p}
		}
	}
	f.Model, cmd = f.Model.Update(msg)
	return f, cmd
}

func createEntryFieldDeclaration(field cuebook.Field, input string) (ast.Decl, error) {
	if field.Value.IncompleteKind()&cue.StringKind != 0 {
		return field.WithValue(input)
	}

	value, err := metadata.FormatAccordingToAttributes(field.Value, input)
	if err != nil {
		return nil, err
	}
	compiled := cuecontext.New().CompileString(value)
	if err := compiled.Err(); err != nil {
		return nil, err
	}
	expression, ok := compiled.Syntax().(ast.Expr)
	if !ok {
		return nil, errors.New("field value is not a CUE expression")
	}
	return &ast.Field{
		Label: ast.NewString(field.Name),
		Value: expression,
	}, nil
}

var createButtonText = &i18n.LocalizeConfig{
	DefaultMessage: &i18n.Message{
		ID:    "bookEntryCreate",
		Other: "Create a New Entry",
	},
}

var createEntryButtonText = &i18n.LocalizeConfig{
	DefaultMessage: &i18n.Message{
		ID:    "bookEntryCreateAction",
		Other: "Create Entry",
	},
}

func NewCreateEntryButton() tea.Model {
	return label.New(list.NewButton(createEntryButtonText, func() tea.Msg {
		return listForm.SaveChangesEvent{}
	}), nil)
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
