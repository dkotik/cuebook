package main

import (
	"context"
	"fmt"
	"os"

	"log/slog"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/terminalui"
	"github.com/dkotik/cuebook/terminalui/card"
	"github.com/dkotik/cuebook/terminalui/field"
	"github.com/dkotik/cuebook/terminalui/file"
	"github.com/dkotik/cuebook/terminalui/list"
	"github.com/dkotik/cuebook/terminalui/textarea"
)

func NewTerminalUI(ctx context.Context, filePath string) tea.Model {
	logFile, err := os.OpenFile("test/testdata/debug.log", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		panic(err)
	}
	go func() {
		<-ctx.Done()
		if err := logFile.Close(); err != nil {
			panic(err)
		}
	}()
	logger := slog.NewTextHandler(logFile, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	window := terminalui.New(
		terminalui.NewEventAdaptor(func(m tea.Model, patch cuebook.SourcePatch) (tea.Model, tea.Cmd) {
			return m, terminalui.WithBusySignal(func() tea.Msg {
				source, err := os.ReadFile(filePath) // TODO: get the path from component
				if err != nil {
					panic(err)
				}
				result, err := patch.Apply(source)
				if err != nil {
					panic(err)
				}
				if err = os.WriteFile(filePath, result.Source, os.FileMode(os.O_CREATE)); err != nil {
					panic(err)
				}
				return result
			})
		})(file.New(filePath)),
		slog.New(logger).With("component", "bubbletea"),
	)

	return terminalui.NewDomainAdaptor(ctx,
		func(ctx context.Context, source file.ContentEvent) (tea.Msg, error) {
			book, err := cuebook.New(source)
			if err != nil {
				return nil, err
			}
			total := book.Len()
			cards := make([]tea.Model, 0, total+1)
			title := list.Title{
				Text:  book.Metadata().Title(),
				Style: lipgloss.NewStyle().Bold(true).Align(lipgloss.Left).Foreground(lipgloss.BrightRed),
			}
			cards = append(cards, title)

			index := 0
			for entry, err := range book.EachEntry() {
				if err != nil {
					return nil, err
				}
				index++
				cards = append(cards, newCardView(ctx, title.View()+fmt.Sprintf(" › %d/%d", index, total), entry, source))
			}
			return terminalui.SwitchTo(list.New(cards...)), nil
		},
		window)
}

func newCardView(
	ctx context.Context,
	title string,
	entry cuebook.Entry,
	source []byte,
) tea.Model {
	return terminalui.NewKeySwitchAdaptor(
		tea.Key{Code: tea.KeyEnter},
		func() tea.Model {
			return newFieldListView(list.Title{
				Text:  title,
				Style: lipgloss.NewStyle().Bold(true).Align(lipgloss.Left).Foreground(lipgloss.BrightRed),
			}, entry, source)
		})(card.New(entry.GetTitle(), entry.GetDescription()...))
}

func newFieldListView(
	title tea.Model,
	entry cuebook.Entry,
	source []byte,
) tea.Model {
	fields := make([]tea.Model, 0, len(entry.Fields)+len(entry.Details)+1)
	fields = append(fields, title)
	for _, f := range entry.Fields {
		fields = append(fields, newFieldView(f, source))
	}
	for _, f := range entry.Details {
		fields = append(fields, newFieldView(f, source))
	}
	return terminalui.NewEventAdaptor(
		func(m tea.Model, source file.ContentEvent) (tea.Model, tea.Cmd) {

			// type entryUpdate struct {
			// 	Field  cuebook.Field
			// 	Range  cuebook.SourceByteRange
			// 	Source []byte
			// }
			type fileUpdate struct {
				Book   cuebook.CueBook
				Source []byte
			}

			// TODO: detect intersection with changed range?
			return m, nil
		})(list.New(fields...))
}

func newFieldView(
	f cuebook.Field,
	source []byte,
) tea.Model {
	closeOnMatchingSourcePatch := terminalui.NewEventAdaptor(func(m tea.Model, r cuebook.SourcePatchResult) (tea.Model, tea.Cmd) {
		if !r.SourceByteRange.IsTouching(cuebook.GetByteSpanInSource(f.Value)) {
			return m, nil
		}
		return m, func() tea.Msg { return terminalui.BackEvent{} }
	})

	return closeOnMatchingSourcePatch(terminalui.NewKeySwitchAdaptor(
		tea.Key{Code: tea.KeyEnter},
		func() tea.Model {
			return terminalui.NewEventAdaptor(
				func(m tea.Model, value textarea.OnChangeEvent) (tea.Model, tea.Cmd) {
					patch, err := f.WithStringValue(source, string(value))
					if err != nil {
						return m, func() tea.Msg { return err }
					}
					return m, func() tea.Msg {
						return patch
					}
					// var crossCommandError error
					// return m, terminalui.WithBusySignal(tea.Sequence(
					// 	func() tea.Msg {
					// 		time.Sleep(time.Second * 4)
					// 		// crossCommandError = errors.New("file operation failed")
					// 		return nil
					// 	},
					// 	func() tea.Msg {
					// 		if crossCommandError != nil {
					// 			return crossCommandError
					// 		}
					// 		return terminalui.BackEvent{}
					// 	},
					// ))
				})(textarea.New(f.Name, f.String(), true))
		})(field.New(f.Name, f.String())))
}
