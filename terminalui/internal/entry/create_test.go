package entry

import (
	"os"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"cuelang.org/go/cue/cuecontext"
	"github.com/charmbracelet/x/exp/teatest/v2"
	"github.com/dkotik/cuebook"
	listForm "github.com/dkotik/cuebook/terminalui/internal/form"
)

func TestCreateFormItemsEndWithCreateAndCancelButtons(t *testing.T) {
	items := createFormItems([]tea.Model{NewCreateButton()})
	if len(items) != 3 {
		t.Fatalf("create form item count = %d, want 3", len(items))
	}
	if button := items[1].View().Content; !strings.Contains(button, "Create Entry") {
		t.Fatalf("penultimate create form item = %q, want Create Entry button", button)
	}
	if button := items[2].View().Content; !strings.Contains(button, "Cancel") {
		t.Fatalf("last create form item = %q, want Cancel button", button)
	}
}

func TestCreateFormUsesSubmittedValues(t *testing.T) {
	source := []byte(`#Entry: {
	Name: string @cuebook(title)
	Age: int
	Enabled?: bool
}
[...#Entry] & [{Name: "Existing", Age: 1}]
`)
	book, err := cuebook.New(source)
	if err != nil {
		t.Fatal(err)
	}
	fields, ok := emitCreateFormFields(book.Value)().(createFormFields)
	if !ok {
		t.Fatalf("emitCreateFormFields() returned %T, want createFormFields", fields)
	}
	document, err := cuebook.New(source)
	if err != nil {
		t.Fatal(err)
	}
	model, _ := newCreateForm(document).Update(fields)
	form, ok := model.(createForm)
	if !ok {
		t.Fatalf("updated model = %T, want createForm", model)
	}

	for _, changed := range []fieldChangedEvent{
		{Name: "Name", Value: "Ada Lovelace"},
		{Name: "Age", Value: "37"},
		{Name: "Enabled", Value: "true"},
	} {
		model, _ = form.Update(changed)
		form = model.(createForm)
	}
	_, cmd := form.Update(listForm.SaveChangesEvent{})
	if cmd == nil {
		t.Fatal("create submission did not produce a command")
	}
	result := cmd()
	patch, ok := result.(createEntryPatch)
	if !ok {
		if err, isError := result.(error); isError {
			t.Fatal(err)
		}
		t.Fatalf("create submission returned %T, want createEntryPatch", result)
	}
	updatedSource, err := patch.ApplyToCueSource(source)
	if err != nil {
		t.Fatal(err)
	}
	updatedBook, err := cuebook.New(updatedSource)
	if err != nil {
		t.Fatal(err)
	}
	created, err := updatedBook.GetEntryByIndex(1)
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{
		"Name":    "Ada Lovelace",
		"Age":     "37",
		"Enabled": "true",
	} {
		field, ok := created.GetFieldByName(name)
		if !ok {
			t.Fatalf("created entry does not contain %q", name)
		}
		if got := field.String(); got != want {
			t.Errorf("created field %q = %q, want %q", name, got, want)
		}
	}
}

func TestCreateFormIgnoresEscape(t *testing.T) {
	document, err := cuebook.New([]byte("[]"))
	if err != nil {
		t.Fatal(err)
	}
	model, cmd := newCreateForm(document).Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if _, ok := model.(createForm); !ok {
		t.Fatalf("updated model = %T, want createForm", model)
	}
	if cmd == nil {
		t.Fatal("Escape returned no command; the window would navigate back")
	}
	if msg := cmd(); msg != nil {
		t.Fatalf("Escape command returned %T, want no message", msg)
	}
}

func TestCreateForm(t *testing.T) {
	t.Skip("fill in later")

	large, err := os.ReadFile("../../../test/testdata/simple.cue")
	if err != nil {
		t.Fatal("unable to read test file")
	}
	book := cuecontext.New().CompileBytes(large)
	if err = book.Err(); err != nil {
		t.Fatal("unable to parse test file:", err)
	}

	tm := teatest.NewTestModel(t, newCreateForm(cuebook.Book{}))
	tm.Send(emitCreateFormFields(book))
	tm.Send(tea.WindowSizeMsg{Width: 20, Height: 20})
	waitForString(t, tm, "Quit? (y/N)")
	// tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
	t.Log("test model", tm)
}

func waitForString(t *testing.T, tm *teatest.TestModel, s string) {
	teatest.WaitFor(
		t,
		tm.Output(),
		func(b []byte) bool {
			return strings.Contains(string(b), s)
		},
		teatest.WithCheckInterval(time.Millisecond*100),
		teatest.WithDuration(time.Second),
	)
}
