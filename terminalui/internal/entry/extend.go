package entry

import (
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/terminalui/event"
	listForm "github.com/dkotik/cuebook/terminalui/internal/form"
	"github.com/dkotik/cuebook/terminalui/internal/label"
	"github.com/dkotik/cuebook/terminalui/internal/list"
	"github.com/dkotik/cuebook/terminalui/window"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// The extend button asks the entry form to start (extendRequestEvent), because
// the entry form knows about unsaved changes. It opens a newFieldForm, which
// collects the name, kind, and value in one place and submits them as an
// addFieldEvent. The entry form then writes the field.
type (
	extendRequestEvent struct{}

	addFieldEvent struct {
		Name  string
		Kind  cuebook.FieldKind
		Value string
	}

	addFieldPatch struct {
		cuebook.Patch
		Name string
	}

	// These have their own types so the entry form below the new field form
	// does not mistake them for changes to its own fields.
	newFieldNameChanged    string
	newFieldKindChanged    cuebook.FieldKind
	newFieldValueChanged   string
	newFieldBooleanChanged string
	newFieldSubmitEvent    struct{}
)

func NewExtendButton() tea.Model {
	return label.New(list.NewButton(
		&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "bookEntryExtend",
				Other: "Add a Custom Field",
			},
		},
		func() tea.Msg { return extendRequestEvent{} },
	), nil)
}

// newFieldForm collects a new field's name, kind, and value. A boolean value
// is chosen from True and False; other values are typed. Both are kept, so
// switching kinds back and forth does not lose what was entered.
type newFieldForm struct {
	tea.Model

	KnownFields []string
	Name        string
	Kind        cuebook.FieldKind
	Value       string
	Boolean     string
	initCmd     tea.Cmd
}

func newNewFieldForm(knownFields []string) newFieldForm {
	f := newFieldForm{
		Model:       listForm.New(listForm.Configuration{}),
		KnownFields: knownFields,
		Kind:        cuebook.FieldKinds()[0],
		Boolean:     "false",
	}
	f, f.initCmd = f.setItemsWithInit(false)
	return f
}

func (f newFieldForm) Init() tea.Cmd {
	return tea.Sequence(f.Model.Init(), f.initCmd)
}

// setItems rebuilds the form rows from the form state. The list keeps its
// selected row.
func (f newFieldForm) setItems() (newFieldForm, tea.Cmd) {
	return f.setItemsWithInit(true)
}

func (f newFieldForm) setItemsWithInit(initialize bool) (newFieldForm, tea.Cmd) {
	items := []tea.Model{
		listForm.NewField("Name", f.Name, func(value string) tea.Cmd { // TODO: localize label
			return func() tea.Msg { return newFieldNameChanged(value) }
		}),
		label.New(newKindSelector(f.Kind), &label.Configuration{Text: "Type"}), // TODO: localize label
		f.valueItem(),
		label.New(list.NewButton(
			&i18n.LocalizeConfig{
				DefaultMessage: &i18n.Message{ID: "bookEntryNewFieldSubmit", Other: "Add Field"},
			},
			func() tea.Msg { return newFieldSubmitEvent{} },
		), nil),
	}
	var initCmd tea.Cmd
	if initialize {
		initCmd = event.PropagateInit(items)
	}
	var setCmd tea.Cmd
	f.Model, setCmd = f.Model.Update(list.SetItems(items...)())
	return f, tea.Sequence(initCmd, setCmd, window.RequestWindowSize(), window.RequestLocalizer())
}

func (f newFieldForm) valueItem() tea.Model {
	if f.Kind == cuebook.FieldKindBoolean {
		return label.New(newBooleanSelector(f.Boolean), &label.Configuration{Text: "Value"}) // TODO: localize label
	}
	return listForm.NewField("Value", f.Value, func(value string) tea.Cmd { // TODO: localize label
		return func() tea.Msg { return newFieldValueChanged(value) }
	})
}

func (f newFieldForm) value() string {
	if f.Kind == cuebook.FieldKindBoolean {
		return f.Boolean
	}
	return f.Value
}

func (f newFieldForm) Update(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	switch msg := msg.(type) {
	case newFieldNameChanged:
		f.Name = string(msg)
		return f, nil
	case newFieldKindChanged:
		wasBoolean := f.Kind == cuebook.FieldKindBoolean
		f.Kind = cuebook.FieldKind(msg)
		if wasBoolean != (f.Kind == cuebook.FieldKindBoolean) {
			return f.setItems()
		}
		return f, nil
	case newFieldValueChanged:
		f.Value = string(msg)
		return f, nil
	case newFieldBooleanChanged:
		f.Boolean = string(msg)
		return f, nil
	case newFieldSubmitEvent:
		return f, f.submit()
	case tea.KeyPressMsg:
		// The list form turns Ctrl+S into a save event, which the entry form
		// below would also act on. Submit this form instead.
		if msg.Key().Code == 's' && msg.Key().Mod.Contains(tea.ModCtrl) {
			return f, f.submit()
		}
	}
	f.Model, cmd = f.Model.Update(msg)
	return f, cmd
}

func (f newFieldForm) submit() tea.Cmd {
	// Surrounding spaces in a field name are almost always accidental.
	name := strings.TrimSpace(f.Name)
	if name == "" || strings.ContainsAny(name, "\r\n") {
		return window.NewFlashMessage(window.FlashMessageKindWarning, &i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "bookEntryInvalidFieldName",
				Other: "A field name must be a single line that is not empty.",
			},
		})
	}
	if slices.Contains(f.KnownFields, name) {
		return window.NewFlashMessage(window.FlashMessageKindWarning, &i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "bookEntryDuplicateFieldName",
				Other: "Field `{{ .Name }}` already exists.",
			},
			TemplateData: map[string]any{"Name": name},
		})
	}
	value := f.value()
	if err := f.Kind.ValidateValue(value); err != nil {
		message := &i18n.Message{ID: "bookEntryInvalidFieldValue", Other: "The value is invalid."}
		if f.Kind == cuebook.FieldKindNumber {
			message = &i18n.Message{ID: "bookEntryInvalidNumberValue", Other: "The value must be a number, such as 42 or -1.5."}
		}
		return window.NewFlashMessage(window.FlashMessageKindWarning, &i18n.LocalizeConfig{DefaultMessage: message})
	}
	added := addFieldEvent{Name: name, Kind: f.Kind, Value: value}
	return func() tea.Msg { return added }
}

func newKindSelector(selected cuebook.FieldKind) optionSelector {
	kinds := cuebook.FieldKinds()
	options := make([]selectorOption, len(kinds))
	for index, kind := range kinds {
		options[index] = selectorOption{
			Value: string(kind),
			Message: &i18n.Message{
				ID:    "bookEntryFieldKind" + strings.ToUpper(string(kind[:1])) + string(kind[1:]),
				Other: kind.Label(),
			},
		}
	}
	return newOptionSelector(options, string(selected), func(value string) tea.Msg {
		return newFieldKindChanged(value)
	})
}

func newBooleanSelector(selected string) optionSelector {
	return newOptionSelector([]selectorOption{
		{Value: "true", Message: &i18n.Message{ID: "bookEntryBooleanTrue", Other: "True"}},
		{Value: "false", Message: &i18n.Message{ID: "bookEntryBooleanFalse", Other: "False"}},
	}, selected, func(value string) tea.Msg {
		return newFieldBooleanChanged(value)
	})
}

type selectorOption struct {
	Value   string
	Message *i18n.Message
}

// optionSelector picks one of a few options on a single line. Left and right,
// or Enter and space, change the option; up and down move to the next form
// row.
type optionSelector struct {
	Options  []selectorOption
	Labels   []string
	Index    int
	Focused  bool
	Width    int
	OnChange func(string) tea.Msg
}

func newOptionSelector(options []selectorOption, selected string, onChange func(string) tea.Msg) optionSelector {
	labels := make([]string, len(options))
	index := 0
	for i, option := range options {
		labels[i] = option.Message.Other
		if option.Value == selected {
			index = i
		}
	}
	return optionSelector{Options: options, Labels: labels, Index: index, OnChange: onChange}
}

func (s optionSelector) Init() tea.Cmd {
	return nil
}

func (s optionSelector) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case *i18n.Localizer:
		// Copy so that selectors rebuilt from the same options do not share
		// a label slice.
		s.Labels = slices.Clone(s.Labels)
		for index, option := range s.Options {
			if text, err := msg.Localize(&i18n.LocalizeConfig{DefaultMessage: option.Message}); err == nil {
				s.Labels[index] = text
			}
		}
	case list.HighlightHintEvent:
		s.Focused = bool(msg)
	case tea.WindowSizeMsg:
		s.Width = msg.Width
	case tea.KeyPressMsg:
		switch msg.Key().Code {
		case tea.KeyRight, tea.KeyEnter, tea.KeySpace, 'l':
			return s.choose((s.Index + 1) % len(s.Options))
		case tea.KeyLeft, 'h':
			return s.choose((s.Index + len(s.Options) - 1) % len(s.Options))
		}
	}
	return s, nil
}

func (s optionSelector) choose(index int) (tea.Model, tea.Cmd) {
	s.Index = index
	value, onChange := s.Options[index].Value, s.OnChange
	return s, func() tea.Msg { return onChange(value) }
}

func (s optionSelector) View() tea.View {
	inactive := lipgloss.NewStyle().Foreground(lipgloss.Color(label.DefaultInactiveColor))
	active := lipgloss.NewStyle()
	if s.Focused {
		active = active.Foreground(lipgloss.Color("73"))
	}
	options := make([]string, len(s.Labels))
	for index, text := range s.Labels {
		if index == s.Index {
			options[index] = active.Render("◉ " + text)
		} else {
			options[index] = inactive.Render("○ " + text)
		}
	}
	return tea.NewView(lipgloss.NewStyle().Width(s.Width).Render(strings.Join(options, "  ")))
}
