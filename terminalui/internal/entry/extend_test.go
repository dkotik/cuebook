package entry

import (
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/dkotik/cuebook"
	listForm "github.com/dkotik/cuebook/terminalui/internal/form"
	"github.com/dkotik/cuebook/terminalui/window"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

func TestLoadFieldsOffersExtendButtonForOpenEntries(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		source string
		want   bool
	}{
		{name: "entry without a schema", source: `[{Name: "Ada"}]`, want: true},
		{name: "open schema", source: "#p: {Name: string, ...}\n[...#p] & [{Name: \"Ada\"}]", want: true},
		{name: "closed schema", source: "#p: {Name: string}\n[...#p] & [{Name: \"Ada\"}]", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, cards := loadEntry(t, tt.source, 0)
			// Every entry ends with the save and delete buttons.
			got := len(cards.Cards) == len(cards.Order)+3
			if got != tt.want {
				t.Errorf("extend button shown = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestFormAddsFieldWithNameKindAndValueFromOneForm(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		// keys are typed into the new field form, which starts on the name.
		keys      []tea.KeyPressMsg
		want      string
		wantOrder []string
	}{
		{
			name: "string submitted with the button",
			keys: keys(typed("Role"), down, down, typed("dev"), down, enter),
			want: `[{Name: "Ada", Role: "dev"}]`, wantOrder: []string{"Name", "Role"},
		},
		{
			name: "multiline string",
			keys: keys(typed("Bio"), down, right, down, typed("one"), enter, typed("two"), ctrlS),
			want: "[{\n\t\tName: \"Ada\"\n\t\tBio: \"\"\"\n\t\t\tone\n\t\t\ttwo\n\t\t\t\"\"\" @cuebook(multiline)\n\t}]", wantOrder: []string{"Name", "Bio"},
		},
		{
			name: "number",
			keys: keys(typed("Age"), down, right, right, down, typed("-1.5"), ctrlS),
			want: `[{Name: "Ada", Age: -1.5}]`, wantOrder: []string{"Name", "Age"},
		},
		{
			name: "boolean defaults to false",
			keys: keys(typed("Active"), down, left, ctrlS),
			want: `[{Name: "Ada", Active: false}]`, wantOrder: []string{"Name", "Active"},
		},
		{
			name: "boolean true is chosen, not typed",
			keys: keys(typed("Active"), down, left, down, right, ctrlS),
			want: `[{Name: "Ada", Active: true}]`, wantOrder: []string{"Name", "Active"},
		},
		{
			name: "boolean ignores text typed before switching kinds",
			keys: keys(typed("Active"), down, down, typed("text"), up, left, down, left, ctrlS),
			want: `[{Name: "Ada", Active: true}]`, wantOrder: []string{"Name", "Active"},
		},
		{
			name: "text survives switching to boolean and back",
			keys: keys(typed("Role"), down, down, typed("dev"), up, left, right, ctrlS),
			want: `[{Name: "Ada", Role: "dev"}]`, wantOrder: []string{"Name", "Role"},
		},
		{
			name: "empty value",
			keys: keys(typed("Age"), down, right, right, ctrlS),
			want: `[{Name: "Ada", Age: 0}]`, wantOrder: []string{"Name", "Age"},
		},
		{
			name: "name is trimmed",
			keys: keys(typed(" Role "), down, down, typed("dev"), ctrlS),
			want: `[{Name: "Ada", Role: "dev"}]`, wantOrder: []string{"Name", "Role"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			state, cards := loadEntry(t, `[{Name: "Ada"}]`, 0)
			entryForm := openForm(t, cards, 0)

			entryForm, cmd := entryForm.Update(extendRequestEvent{})
			msgs := runCommand(cmd)
			if len(msgs) != 1 {
				t.Fatalf("extend request produced %#v, want the new field form", msgs)
			}
			newForm, ok := msgs[0].(newFieldForm)
			if !ok {
				t.Fatalf("extend request opened %T, want the new field form", msgs[0])
			}
			if !slices.Equal(newForm.KnownFields, []string{"Name"}) {
				t.Errorf("known fields = %q, want the entry's fields", newForm.KnownFields)
			}

			var model tea.Model = newForm
			cmd = model.Init()
			model, _ = feed(model, runWindowCommand(cmd))
			var emitted []tea.Msg
			for _, key := range tt.keys {
				var out []tea.Msg
				model, cmd = model.Update(key)
				model, out = feed(model, runWindowCommand(cmd))
				emitted = append(emitted, out...)
			}
			var added *addFieldEvent
			for _, msg := range emitted {
				switch msg := msg.(type) {
				case addFieldEvent:
					added = &msg
				case listForm.SaveChangesEvent:
					t.Fatal("new field form emitted a save event that the entry form would act on")
				}
			}
			if added == nil {
				t.Fatalf("new field form did not submit; emitted %#v", emitted)
			}

			entryForm, cmd = entryForm.Update(*added)
			msgs = runCommand(cmd)
			if len(msgs) != 2 || msgs[0] != (window.BackEvent{}) {
				t.Fatalf("add field produced %#v, want to close the form and update the entry", msgs)
			}
			update, ok := msgs[1].(Update)
			if !ok {
				t.Fatalf("add field produced %T, want an entry update", msgs[1])
			}
			patch, err := update(state)
			if err != nil {
				t.Fatal(err)
			}
			updated, err := patch.ApplyToCueSource(state.Source)
			if err != nil {
				t.Fatal(err)
			}
			if string(updated) != tt.want {
				t.Fatalf("source =\n%s\nwant\n%s", updated, tt.want)
			}

			book, err := cuebook.New(updated)
			if err != nil {
				t.Fatal(err)
			}
			_, cmd = entryForm.Update(cuebook.Result{Document: book, Source: updated, LastChange: patch})
			var reloaded *fieldListCards
			for _, msg := range runCommand(cmd) {
				if cards, ok := msg.(fieldListCards); ok {
					reloaded = &cards
				}
			}
			if reloaded == nil {
				t.Fatal("result did not reload the fields")
			}
			if got := orderNames(reloaded.Order); !slices.Equal(got, tt.wantOrder) {
				t.Errorf("reloaded order = %q, want %q", got, tt.wantOrder)
			}
			if got := reloaded.Order[reloaded.SelectedIndex].Name; got != tt.wantOrder[len(tt.wantOrder)-1] {
				t.Errorf("selected field = %q, want the new field", got)
			}
		})
	}
}

func TestNewFieldFormRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		fieldName string
		kind      cuebook.FieldKind
		value     string
	}{
		{name: "empty name", fieldName: " \n ", kind: cuebook.FieldKindString},
		{name: "name with several lines", fieldName: "Role\nOwner", kind: cuebook.FieldKindString},
		{name: "existing field", fieldName: "Name", kind: cuebook.FieldKindString},
		{name: "text as number", fieldName: "Age", kind: cuebook.FieldKindNumber, value: "ten"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			form := newNewFieldForm([]string{"Name"})
			form.Name, form.Kind, form.Value = tt.fieldName, tt.kind, tt.value
			_, cmd := form.Update(newFieldSubmitEvent{})
			msgs := runCommand(cmd)
			if len(msgs) == 0 {
				t.Fatal("invalid input produced no warning")
			}
			for _, msg := range msgs {
				if _, ok := msg.(addFieldEvent); ok {
					t.Fatalf("invalid input was submitted: %+v", msg)
				}
			}
		})
	}
}

func TestNewFieldFormShowsBooleanChoicesOnlyForBooleans(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		kind cuebook.FieldKind
		want bool
	}{
		{name: "string", kind: cuebook.FieldKindString},
		{name: "multiline string", kind: cuebook.FieldKindMultiLine},
		{name: "number", kind: cuebook.FieldKindNumber},
		{name: "boolean", kind: cuebook.FieldKindBoolean, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var model tea.Model = newNewFieldForm(nil)
			cmd := model.Init()
			model, _ = feed(model, runWindowCommand(cmd))
			model, cmd = model.Update(newFieldKindChanged(tt.kind))
			model, _ = feed(model, runWindowCommand(cmd))
			view := model.View().Content
			if got := strings.Contains(view, "○ True") && strings.Contains(view, "◉ False"); got != tt.want {
				t.Errorf("boolean choices shown = %t, want %t:\n%s", got, tt.want, view)
			}
		})
	}
}

func TestNewFieldFormRendersBeforeItKnowsTheWindowSize(t *testing.T) {
	t.Parallel()
	var model tea.Model = newNewFieldForm([]string{"Name"})
	_ = model.Init()
	_ = model.View()
}

func TestFormRequiresSavedChangesBeforeAddingAField(t *testing.T) {
	t.Parallel()
	_, cards := loadEntry(t, `[{Name: "Ada"}]`, 0)
	model := openForm(t, cards, 0)
	model, _ = model.Update(fieldChangedEvent{Name: "Name", Value: "Grace", Original: "Ada"})
	_, cmd := model.Update(extendRequestEvent{})
	for _, msg := range runCommand(cmd) {
		if _, ok := msg.(newFieldForm); ok {
			t.Fatal("new field form opened with unsaved changes")
		}
	}
}

var (
	down  = tea.KeyPressMsg{Code: tea.KeyDown}
	up    = tea.KeyPressMsg{Code: tea.KeyUp}
	left  = tea.KeyPressMsg{Code: tea.KeyLeft}
	right = tea.KeyPressMsg{Code: tea.KeyRight}
	enter = tea.KeyPressMsg{Code: tea.KeyEnter}
	ctrlS = tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl}
)

func typed(text string) []tea.KeyPressMsg {
	presses := make([]tea.KeyPressMsg, 0, len(text))
	for _, r := range text {
		presses = append(presses, tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	return presses
}

func keys(groups ...any) []tea.KeyPressMsg {
	var presses []tea.KeyPressMsg
	for _, group := range groups {
		switch group := group.(type) {
		case tea.KeyPressMsg:
			presses = append(presses, group)
		case []tea.KeyPressMsg:
			presses = append(presses, group...)
		}
	}
	return presses
}

// feed delivers messages that the new field form sends to itself, and the
// window's answers to its requests, as the window would. It returns the other
// messages.
func feed(model tea.Model, msgs []tea.Msg) (tea.Model, []tea.Msg) {
	var others []tea.Msg
	for len(msgs) > 0 {
		msg := msgs[0]
		msgs = msgs[1:]
		switch msg.(type) {
		case newFieldNameChanged, newFieldKindChanged, newFieldValueChanged, newFieldBooleanChanged, newFieldSubmitEvent,
			tea.WindowSizeMsg, *i18n.Localizer:
			var cmd tea.Cmd
			model, cmd = model.Update(msg)
			msgs = append(msgs, runWindowCommand(cmd)...)
		default:
			others = append(others, msg)
		}
	}
	return model, others
}

// runCommand is [runWindowCommand] without the window's answers.
func runCommand(cmd tea.Cmd) []tea.Msg {
	var msgs []tea.Msg
	for _, msg := range runWindowCommand(cmd) {
		switch msg.(type) {
		case tea.WindowSizeMsg, *i18n.Localizer:
		default:
			msgs = append(msgs, msg)
		}
	}
	return msgs
}

// runWindowCommand runs cmd and the commands of any batch or sequence it
// returns, in order, and collects the messages. Requests for a window size or
// a localizer are replaced with the window's answers, so that text does not
// wrap at every character and labels are localized. Commands that wait, such
// as cursor blinks, never return promptly and are dropped.
func runWindowCommand(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	result := make(chan tea.Msg, 1)
	go func() { result <- cmd() }()
	var msg tea.Msg
	select {
	case msg = <-result:
	case <-time.After(50 * time.Millisecond):
		return nil
	}
	if msg == nil {
		return nil
	}
	// tea.Sequence returns an unexported slice of commands.
	if value := reflect.ValueOf(msg); value.Kind() == reflect.Slice && value.Type().Elem() == reflect.TypeFor[tea.Cmd]() {
		var msgs []tea.Msg
		for index := range value.Len() {
			msgs = append(msgs, runWindowCommand(value.Index(index).Interface().(tea.Cmd))...)
		}
		return msgs
	}
	switch reflect.TypeOf(msg).String() {
	case "tea.windowSizeMsg":
		return []tea.Msg{tea.WindowSizeMsg{Width: 120, Height: 40}}
	case "window.localizerRequestEvent":
		return []tea.Msg{i18n.NewLocalizer(i18n.NewBundle(language.AmericanEnglish), "en")}
	}
	return []tea.Msg{msg}
}
