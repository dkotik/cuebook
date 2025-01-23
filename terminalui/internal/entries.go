package internal

import (
	"bytes"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/cuecontext"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/metadata"
	"github.com/dkotik/cuebook/patch"
	"github.com/dkotik/cuebook/terminalui/card"
	"github.com/dkotik/cuebook/terminalui/event"
	"github.com/dkotik/cuebook/terminalui/list"
	"github.com/dkotik/cuebook/terminalui/window"
)

type entryHighlighted int

type EntryList struct {
	tea.Model

	book     patch.Result
	selected int // *patch.ByteAnchor
}

func (l EntryList) Init() (_ tea.Model, cmd tea.Cmd) {
	l.Model = list.New()
	l.selected = -1
	return l, nil
}

type entryListCards struct {
	Cards         []tea.Model
	SelectedIndex int
}

func (l EntryList) LoadEntry(index int) tea.Cmd {
	return func() tea.Msg {
		entry, err := cuebook.NewEntry(l.book.Document.LookupPath(cue.MakePath(cue.Index(index))))
		if err != nil {
			return err // TODO: fails on empty list?
		}
		return entry
	}
}

func (l EntryList) Update(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	switch msg := msg.(type) {
	case entryHighlighted:
		l.selected = int(msg)
		return l, nil
	case patch.Patch:
		return l, func() tea.Msg { // TODO: handle switch highlight
			result, err := patch.Commit("test/testdata/simple.cue", "test/testdata", msg)
			if err != nil {
				return err
			}
			return result
		}
	case patch.Result:
		switch p := msg.LastChange.(type) {
		case swapEntryPatch:
			target, err := p.Difference().Match(msg.Source)
			if err == nil {
				i := 0
				for entry := range msg.Document.EachValue() {
					i++
					r, err := patch.NewByteRange(entry)
					if err == nil && r == target {
						l.selected = i
						break
					}
				}
			}
		}

		// if l.book.IsEqual(msg) {
		// 	return l, nil
		// }
		l.book = msg
		return l, LoadEntries(msg, l.selected)
	case entryListCards:
		l.selected = msg.SelectedIndex
		l.Model, cmd = l.Model.Init()
		var setCmd, updateCmd tea.Cmd
		l.Model, setCmd = l.Model.Update(list.SetItems(msg.Cards...)())
		l.Model, updateCmd = l.Model.Update(list.ApplySelection(l.selected)())
		return l, tea.Sequence(cmd, setCmd, tea.RequestWindowSize(), updateCmd, l.LoadEntry(msg.SelectedIndex))
	case entrySelected:
		// l.selected = int(msg) + 1
		return l, tea.Sequence(
			func() tea.Msg {
				return window.SwitchTo(FieldList{state: l.book})
			},
			l.LoadEntry(int(msg)),
		)
	case list.SwapOrderEvent:
		return l, func() tea.Msg {
			a, err := cuebook.NewEntry(l.book.Document.LookupPath(cue.MakePath(cue.Index(msg.CurrentIndex - 1))))
			if err != nil {
				return err
			}
			b, err := cuebook.NewEntry(l.book.Document.LookupPath(cue.MakePath(cue.Index(msg.DesiredIndex - 1))))
			if err != nil {
				return err
			}
			p, err := patch.SwapEntries(l.book.Source, a.Value, b.Value)
			if err != nil {
				return err
			}
			// target, err := patch.NewByteRange(a.Value)
			// if err != nil {
			// 	return err
			// }
			return swapEntryPatch{Patch: p}
		}
	case tea.KeyMsg:
		switch msg.Key().Code {
		case 'x':
			if msg.Key().Mod != tea.ModCtrl {
				break
			}
			index, err := l.book.Document.Len()
			if err != nil {
				panic(err)
			}
			index -= 1
			l.selected--
			return l, func() tea.Msg {
				entry, err := cuebook.NewEntry(l.book.Document.LookupPath(cue.MakePath(cue.Index(index))))
				if err != nil {
					return err
				}
				p, err := patch.DeleteFromStructList(
					l.book.Source, entry.Value)
				if err != nil {
					return err
				}
				return p
			}
		case 'n':
			if msg.Key().Mod != tea.ModCtrl {
				break
			}
			return l, func() tea.Msg {
				value := cuecontext.New().BuildExpr(
					ast.NewStruct(
						&ast.Field{
							Label: ast.NewString("Name"),
							Value: ast.NewString("Someone"),
						},
						&ast.Field{
							Label: ast.NewString("Email"),
							Value: ast.NewString("someEmail@somehost.net"),
						},
					),
				)
				p, err := patch.AppendToStructList(l.book.Source, value)
				if err != nil {
					return err
				}
				return p
			}
		}
		l.Model, cmd = l.Model.Update(msg)
		return l, cmd
	}
	l.Model, cmd = l.Model.Update(msg)
	return l, cmd
}

func LoadEntries(r patch.Result, selectionIndex int) tea.Cmd {
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

		title := event.NewAdaptor[tea.KeyMsg](func(m tea.Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
			switch msg.Key().Code {
			case tea.KeyEnter:
				return m, tea.Sequence(
					func() tea.Msg {
						return window.SwitchTo(FrontMatterView{})
					},
					func() tea.Msg {
						return r // to populate the frontmatter view
					},
				)
			default:
				return m, nil
			}
		})(list.Title{
			Text:  metadata.NewFrontmatter(r.Source).Title(),
			Style: lipgloss.NewStyle().Bold(true).Align(lipgloss.Left).Foreground(lipgloss.BrightRed),
		})
		result.Cards = append(result.Cards, title)

		var (
			index                          int
			lastChangePreceedingDuplicates int
			lastChange                     []byte
		)
		if r.LastChange != nil {
			diff := r.LastChange.Difference()
			lastChange = diff.Content
			if firstBrace := bytes.IndexByte(lastChange, '{'); firstBrace > 0 && firstBrace < 10 {
				lastChange = lastChange[firstBrace:]
			}
			if lastBrace := bytes.LastIndexByte(lastChange, '}'); lastBrace > 0 && lastBrace > len(lastChange)-10 {
				lastChange = lastChange[:lastBrace+1]
			}
			lastChangePreceedingDuplicates = diff.PreceedingDuplicates
		}
		for entry, err := range r.Document.EachEntry() {
			if err != nil {
				return err
			}
			if lastChange != nil {
				at, err := patch.NewByteRange(entry.Value)
				if err != nil {
					return err
				}
				if lastChangePreceedingDuplicates < 0 {
					lastChange = nil // stop tracking
				}
				if bytes.Equal(lastChange, r.Source[at.Head:at.Tail]) {
					result.SelectedIndex = index
					lastChangePreceedingDuplicates--
				}
			}
			result.Cards = append(
				result.Cards,
				Entry{
					Model: card.New(entry.GetTitle(), entry.GetDescription()...),
					Index: index,
				},
			)
			index++
		}
		return result
	}
}
