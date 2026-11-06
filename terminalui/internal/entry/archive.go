package entry

import (
	tea "charm.land/bubbletea/v2"
	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/terminalui/internal/label"
	"github.com/dkotik/cuebook/terminalui/internal/list"
	"github.com/dkotik/cuebook/terminalui/window"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

var archiveButtonText = &i18n.LocalizeConfig{
	DefaultMessage: &i18n.Message{
		ID:    "bookEntryArchive",
		Other: "Archive Entry",
	},
}

type archiveEvent struct{}

// ArchiveEntryEvent requests moving an entry to the daily archive.
type ArchiveEntryEvent struct {
	Entry        cuebook.Entry
	ReturnToList bool
}

// ArchiveEntryCompletedEvent reports a successful archive operation.
type ArchiveEntryCompletedEvent struct {
	Title        string
	Result       cuebook.Result
	ReturnToList bool
}

func NewArchiveButton() tea.Model {
	return label.New(archiveButton{
		Model: list.NewButton(archiveButtonText, func() tea.Msg {
			return archiveEvent{}
		}),
		Cancel: NewCancelButton(),
	}, nil)
}

type archiveButton struct {
	tea.Model
	Cancel     tea.Model
	ShowCancel bool
}

func (d archiveButton) Init() tea.Cmd {
	return tea.Batch(d.Model.Init(), d.Cancel.Init())
}

func (d archiveButton) Update(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	switch msg.(type) {
	case fieldChangedEvent:
		d.ShowCancel = true
		return d, tea.Batch(window.RequestLocalizer(), window.RequestWindowSize())
	case cancelEvent:
		d.ShowCancel = false
		return d, tea.Batch(window.RequestLocalizer(), window.RequestWindowSize())
	}

	if d.ShowCancel {
		d.Cancel, cmd = d.Cancel.Update(msg)
	} else {
		d.Model, cmd = d.Model.Update(msg)
	}
	return d, cmd
}

func (d archiveButton) View() tea.View {
	if d.ShowCancel {
		return d.Cancel.View()
	}
	return d.Model.View()
}
