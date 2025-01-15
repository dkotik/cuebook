package internal

import (
	"bytes"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/cuecontext"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/patch"
	"github.com/dkotik/cuebook/terminalui/card"
	"github.com/dkotik/cuebook/terminalui/list"
	"github.com/dkotik/cuebook/terminalui/window"
)

type (
	entrySelected int

	entryAdded struct {
		patch.Patch
		UpdatedSource []byte
	}
)

type EntryList struct {
	tea.Model

	book     Book
	selected int // *patch.ByteAnchor
}

func (l EntryList) Init() (_ tea.Model, cmd tea.Cmd) {
	l.Model = list.New()
	l.selected = -1
	return l, nil
}

type entryListCards []tea.Model

// func (l EntryList)

func (l EntryList) Update(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	switch msg := msg.(type) {
	case entrySelected:
		l.selected = int(msg)
		return l, func() tea.Msg {
			return tea.BatchMsg{
				tea.Sequence(
					func() tea.Msg {
						return window.SwitchTo(FieldList{
							book: l.book,
						})
					},
					func() tea.Msg {
						entry, err := cuebook.NewEntry(l.book.Document.LookupPath(cue.MakePath(cue.Index(l.selected))))
						if err != nil {
							return err
						}
						return entry
					},
				),
			}
		}
	case Book:
		l.book = msg
		return l, LoadEntries(msg.Document, l.selected, nil) // TODO: track patch changes
	case entryListCards:
		l.Model, cmd = l.Model.Init()
		var setCmd tea.Cmd
		l.Model, setCmd = l.Model.Update(list.SetItems(msg...)())
		return l, tea.Sequence(cmd, setCmd, tea.RequestWindowSize())
	case tea.KeyMsg:
		switch msg.Key().Code {
		case tea.KeyEnter:
			l.Model, cmd = l.Model.Update(msg)
			return l, NewSelectionAdapter[entrySelected](cmd)
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
				result, err := p.ApplyToCueSource(l.book.Source)
				if err != nil {
					return err
				}
				return entryAdded{
					Patch:         p,
					UpdatedSource: result,
				}
			}
		}
	}
	l.Model, cmd = l.Model.Update(msg)
	return l, cmd
}

func LoadEntries(book cuebook.Document, currentSelection int, r *cuebook.SourcePatchResult) tea.Cmd {
	return func() tea.Msg {
		total, err := book.Len()
		if err != nil {
			return err
		}
		cards := make([]tea.Model, 0, total+1)
		title := list.Title{
			Text:  book.Metadata().Title(),
			Style: lipgloss.NewStyle().Bold(true).Align(lipgloss.Left).Foreground(lipgloss.BrightRed),
		}
		cards = append(cards, title)

		index := 0
		selectIndex := -1
		for entry, err := range book.EachEntry() {
			if err != nil {
				return err
			}
			if r != nil && r.PrecedingDuplicates >= 0 {
				at := cuebook.GetByteSpanInSource(entry.Value)
				if !at.IsValid() {
					continue // TODO: handle
				}
				if bytes.Equal(r.ReplaceWith, r.Source[at.BeginsAt:at.EndsAt]) {
					selectIndex = index
					r.PrecedingDuplicates--
				}
				index++
			}
			cards = append(cards, card.New(entry.GetTitle(), entry.GetDescription()...))
		}
		if selectIndex >= 0 { // found matching bytes
			currentSelection = selectIndex // TODO: write a test for it
		}
		return entryListCards(cards)
	}
}
