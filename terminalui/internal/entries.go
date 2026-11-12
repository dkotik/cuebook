package internal

import (
	"cmp"
	"context"
	"errors"
	"path/filepath"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/internal/markdown"
	"github.com/dkotik/cuebook/metadata"
	"github.com/dkotik/cuebook/terminalui/event"
	"github.com/dkotik/cuebook/terminalui/internal/card"
	"github.com/dkotik/cuebook/terminalui/internal/entry"
	"github.com/dkotik/cuebook/terminalui/internal/list"
	"github.com/dkotik/cuebook/terminalui/window"
)

const (
	booleanTrue            = "✓"
	informationUnavailable = " ⃠"
)

type entryHighlighted int

type EntryList struct {
	tea.Model

	sourcePath string
	book       cuebook.Result
	selected   int // *cuebook.ByteAnchor
}

func NewEntryList(sourcePath string) tea.Model {
	return EntryList{
		Model:      list.New(list.Configuration{}),
		sourcePath: sourcePath,
		selected:   -1,
	}
}

func (l EntryList) sourceFileSystem() (cuebook.FileSystem, string, error) {
	if l.sourcePath == "" {
		return nil, "", errors.New("source file path is empty")
	}
	path := filepath.Clean(l.sourcePath)
	name := filepath.Base(path)
	if name == "." || name == string(filepath.Separator) {
		return nil, "", errors.New("source path does not name a file")
	}
	files, err := cuebook.NewFileSystem(cuebook.FileSystemConfig{Root: filepath.Dir(path)})
	if err != nil {
		return nil, "", err
	}
	return files, name, nil
}

func (l EntryList) Init() tea.Cmd {
	return l.Model.Init()
}

type entryListCards struct {
	Cards         []tea.Model
	SelectedIndex int
}

func (l EntryList) Update(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	switch msg := msg.(type) {
	case entryHighlighted:
		l.selected = int(msg)
		return l, nil
	case entry.Update:
		return l, func() tea.Msg {
			patch, err := msg(l.book)
			if err != nil {
				return err
			}
			return patch
		}
	case entry.ArchiveEntryEvent:
		return l, func() tea.Msg {
			files, sourceName, err := l.sourceFileSystem()
			if err != nil {
				return err
			}
			result, err := archiveEntry(context.Background(), files, sourceName, msg.Book, msg.Entry, time.Now())
			if err != nil {
				return err
			}
			return entry.ArchiveEntryCompletedEvent{
				Title:        msg.Entry.Entry().Title,
				Result:       result,
				ReturnToList: msg.ReturnToList,
			}
		}
	case entry.ArchiveEntryCompletedEvent:
		l.book = msg.Result
		refresh := LoadEntries(msg.Result, l.selected)
		if msg.ReturnToList {
			return l, tea.Sequence(refresh, func() tea.Msg { return window.BackEvent{} })
		}
		return l, refresh
	case cuebook.Patch:
		return l, func() tea.Msg { // TODO: handle switch highlight
			files, sourceName, err := l.sourceFileSystem()
			if err != nil {
				return err
			}
			result, err := cuebook.Commit(context.Background(), files, sourceName, msg)
			if err != nil {
				return err
			}
			return result
		}
	case cuebook.Result:
		// if l.book.IsEqual(msg) {
		// 	return l, nil
		// }
		l.book = msg
		if msg.LastChange != nil {
			return l, LoadEntries(msg, l.selected)
		}
		return l, LoadEntries(msg, l.selected)
	case entryListCards:
		l.selected = msg.SelectedIndex
		cmd = l.Model.Init()
		var setCmd, updateCmd tea.Cmd
		l.Model, setCmd = l.Model.Update(list.SetItems(msg.Cards...)())
		l.Model, updateCmd = l.Model.Update(list.ApplySelection(l.selected)())
		return l, tea.Sequence(
			cmd,
			setCmd,
			window.RequestWindowSize(),
			window.RequestLocalizer(),
			updateCmd,
		)
	case entry.CreateEvent:
		return l, entry.NewCreateForm(l.book.Document)
	case entrySelected:
		selected, err := l.book.Document.GetEntryRef(int(msg))
		if err != nil {
			return l, func() tea.Msg { return err } // TODO: fails on empty list?
		}
		return l, entry.NewForm(l.book.Document, selected)
	case list.SwapOrderEvent:
		if msg.CurrentIndex < 1 || msg.DesiredIndex < 1 {
			return l, nil // the title card at index 0 is not an entry
		}
		return l, func() tea.Msg {
			a, err := l.book.Document.GetEntryRef(msg.CurrentIndex - 1)
			if err != nil {
				return nil // does not exist
			}
			b, err := l.book.Document.GetEntryRef(msg.DesiredIndex - 1)
			if err != nil {
				return nil // does not exist
			}
			p, err := l.book.Document.SwapEntries(a, b)
			if err != nil {
				return err
			}
			return p
		}
	case tea.KeyPressMsg:
		switch msg.Key().Code {
		case 'x':
			if msg.Key().Mod != tea.ModCtrl {
				break
			}
			index, err := l.book.Document.Len()
			if err != nil {
				panic(err)
			}
			index--
			selectedEntry, err := l.book.Document.GetEntryRef(index)
			if err != nil {
				return l, func() tea.Msg { return err }
			}
			l.selected--
			return l, func() tea.Msg { return entry.ArchiveEntryEvent{Book: l.book.Document, Entry: selectedEntry} }
		case 'n':
			if msg.Key().Mod != tea.ModCtrl {
				break
			}
			return l, entry.NewCreateForm(l.book.Document)
		}
		l.Model, cmd = l.Model.Update(msg)
		return l, cmd
	}
	l.Model, cmd = l.Model.Update(msg)
	return l, cmd
}

func LoadEntries(r cuebook.Result, selectionIndex int) tea.Cmd {
	return func() tea.Msg {
		total, err := r.Document.Len()
		if err != nil {
			return err
		}
		if selectionIndex > total {
			selectionIndex = total - 1
		}
		result := entryListCards{
			Cards:         make([]tea.Model, 0, total+2),
			SelectedIndex: selectionIndex,
		}

		title := event.NewAdaptor[tea.KeyPressMsg](func(m tea.Model, msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
			switch msg.Key().Code {
			case tea.KeyEnter:
				return m, tea.Sequence(
					func() tea.Msg {
						return window.SwitchTo(newFrontMatterView())
					},
					func() tea.Msg {
						return r // to populate the frontmatter view
					},
				)
			default:
				return m, nil
			}
		})(list.Title{
			Text:  cmp.Or(metadata.NewFrontmatter(r.Source, markdown.NewParser()).Title(), informationUnavailable),
			Style: lipgloss.NewStyle().Bold(true).Align(lipgloss.Left).Foreground(lipgloss.BrightRed),
		})
		result.Cards = append(result.Cards, title)

		var (
			index         int
			selectedRange cuebook.ByteRange
			hasSelection  bool
		)
		if r.LastChange != nil {
			if selected, ok := r.Match(r.LastChange.Difference().Content); ok {
				selectedRange, err = cuebook.NewByteRange(selected.Value)
				hasSelection = err == nil
			}
		}
		for entry, err := range r.Document.EachEntry() {
			if err != nil {
				return err
			}
			if hasSelection {
				at, err := cuebook.NewByteRange(entry.Value)
				if err != nil {
					return err
				}
				if at == selectedRange {
					result.SelectedIndex = index + 1
				}
			}
			result.Cards = append(
				result.Cards,
				Entry{
					Model: card.New(card.Configuration{
						Title:       entry.Title,
						Description: entry.GetDescription(),
					}),
					Index: index,
				},
			)
			index++
		}
		result.Cards = append(
			result.Cards,
			entry.NewCreateButton(),
		)
		return result
	}
}
