package entry

import (
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
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
	newFieldNameChanged  string
	newFieldKindChanged  cuebook.FieldKind
	newFieldValueChanged string
	newFieldSubmitEvent  struct{}
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

// newFieldForm collects a new field's name, kind, and value.
type newFieldForm struct {
	tea.Model

	KnownFields []string
	Name        string
	Kind        cuebook.FieldKind
	Value       string
}

func newNewFieldForm(knownFields []string) newFieldForm {
	return newFieldForm{
		Model:       listForm.New(listForm.Configuration{}),
		KnownFields: knownFields,
		Kind:        cuebook.FieldKinds()[0],
	}
}

func (f newFieldForm) Init() (_ tea.Model, cmd tea.Cmd) {
	items := []tea.Model{
		listForm.NewField("Name", "", func(value string) tea.Cmd { // TODO: localize label
			return func() tea.Msg { return newFieldNameChanged(value) }
		}),
		label.New(newKindSelector(f.Kind), &label.Configuration{Text: "Type"}), // TODO: localize label
		listForm.NewField("Value", "", func(value string) tea.Cmd { // TODO: localize label
			return func() tea.Msg { return newFieldValueChanged(value) }
		}),
		label.New(list.NewButton(
			&i18n.LocalizeConfig{
				DefaultMessage: &i18n.Message{ID: "bookEntryNewFieldSubmit", Other: "Add Field"},
			},
			func() tea.Msg { return newFieldSubmitEvent{} },
		), nil),
	}
	f.Model, cmd = f.Model.Init()
	initCmd := event.PropagateInit(items)
	var setCmd tea.Cmd
	f.Model, setCmd = f.Model.Update(list.SetItems(items...)())
	return f, tea.Sequence(cmd, initCmd, setCmd, tea.RequestWindowSize(), window.RequestLocalizer())
}

func (f newFieldForm) Update(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	switch msg := msg.(type) {
	case newFieldNameChanged:
		f.Name = string(msg)
		return f, nil
	case newFieldKindChanged:
		f.Kind = cuebook.FieldKind(msg)
		return f, nil
	case newFieldValueChanged:
		f.Value = string(msg)
		return f, nil
	case newFieldSubmitEvent:
		return f, f.submit()
	case tea.KeyMsg:
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
	if err := f.Kind.ValidateValue(f.Value); err != nil {
		message := &i18n.Message{ID: "bookEntryInvalidFieldValue", Other: "The value is invalid."}
		switch f.Kind {
		case cuebook.FieldKindNumber:
			message = &i18n.Message{ID: "bookEntryInvalidNumberValue", Other: "The value must be a number, such as 42 or -1.5."}
		case cuebook.FieldKindBoolean:
			message = &i18n.Message{ID: "bookEntryInvalidBooleanValue", Other: "The value must be true or false."}
		}
		return window.NewFlashMessage(window.FlashMessageKindWarning, &i18n.LocalizeConfig{DefaultMessage: message})
	}
	added := addFieldEvent{Name: name, Kind: f.Kind, Value: f.Value}
	return func() tea.Msg { return added }
}

var fieldKindMessages = map[cuebook.FieldKind]*i18n.Message{
	cuebook.FieldKindString:    {ID: "bookEntryFieldKindString", Other: "String"},
	cuebook.FieldKindMultiLine: {ID: "bookEntryFieldKindMultiLine", Other: "Multiline String"},
	cuebook.FieldKindNumber:    {ID: "bookEntryFieldKindNumber", Other: "Number"},
	cuebook.FieldKindBoolean:   {ID: "bookEntryFieldKindBoolean", Other: "Boolean"},
}

// kindSelector picks one of the field kinds. Left and right, or Enter and
// space, change the kind; up and down move to the next form item.
type kindSelector struct {
	Kinds   []cuebook.FieldKind
	Labels  []string
	Index   int
	Focused bool
	Width   int
}

func newKindSelector(selected cuebook.FieldKind) kindSelector {
	kinds := cuebook.FieldKinds()
	labels := make([]string, len(kinds))
	for index, kind := range kinds {
		labels[index] = kind.Label()
	}
	return kindSelector{
		Kinds:  kinds,
		Labels: labels,
		Index:  max(0, slices.Index(kinds, selected)),
	}
}

func (s kindSelector) Init() (tea.Model, tea.Cmd) {
	return s, nil
}

func (s kindSelector) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case *i18n.Localizer:
		for index, kind := range s.Kinds {
			if message, ok := fieldKindMessages[kind]; ok {
				if text, err := msg.Localize(&i18n.LocalizeConfig{DefaultMessage: message}); err == nil {
					s.Labels[index] = text
				}
			}
		}
	case list.HighlightHintEvent:
		s.Focused = bool(msg)
	case tea.WindowSizeMsg:
		s.Width = msg.Width
	case tea.KeyMsg:
		switch msg.Key().Code {
		case tea.KeyRight, tea.KeyEnter, tea.KeySpace, 'l':
			return s.choose((s.Index + 1) % len(s.Kinds))
		case tea.KeyLeft, 'h':
			return s.choose((s.Index + len(s.Kinds) - 1) % len(s.Kinds))
		}
	}
	return s, nil
}

func (s kindSelector) choose(index int) (tea.Model, tea.Cmd) {
	s.Index = index
	kind := s.Kinds[index]
	return s, func() tea.Msg { return newFieldKindChanged(kind) }
}

func (s kindSelector) View() string {
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
	return lipgloss.NewStyle().Width(s.Width).Render(strings.Join(options, "  "))
}
