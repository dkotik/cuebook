package main

import (
	"context"
	"fmt"
	"sync"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/terminalui"
	"github.com/dkotik/cuebook/terminalui/card"
	"github.com/dkotik/cuebook/terminalui/file"
	"github.com/dkotik/cuebook/terminalui/list"
)

type cueFileState struct {
	Tree   cue.Value
	Source []byte
	mu     sync.Mutex
}

func (s *cueFileState) Load(ctx context.Context, content file.ContentEvent) (tea.Msg, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.parse(content); err != nil {
		return nil, err
	}
	size, err := s.Tree.Len().Int64()
	if err != nil {
		return nil, fmt.Errorf("unable to get the length of the Cue list: %w", err)
	}

	cards := make([]tea.Model, 0, int(size))
	// next, err := s.Tree.List()
	// if err != nil {
	// 	return nil, errors.New("unable to iterate over the Cue list")
	// }
	// for next.Next() {
	// 	cards = append(cards, card.New("title string"+strconv.Itoa(len(cards)), "description ...string"))
	// }
	book, err := cuebook.New(content)
	if err != nil {
		return nil, err
	}
	for entry, err := range book.EachEntry() {
		if err != nil {
			return nil, err
		}
		cards = append(cards, card.New(entry.String(), "description ...string"))
	}

	return terminalui.SwitchTo(list.New(cards...)), nil
}

func (s *cueFileState) parse(source []byte) (err error) {
	s.Tree = cuecontext.New().CompileBytes(source)
	err = s.Tree.Err()
	if err != nil {
		return fmt.Errorf("unable to parse Cue list: %w", err)
	}
	if err = s.Tree.Validate(
		cue.Attributes(true),
		cue.Concrete(true),
		cue.Definitions(true),
	); err != nil {
		return fmt.Errorf("unable to validate Cue list: %w", err)
	}
	if s.Tree.Kind() != cue.ListKind {
		return fmt.Errorf("expected a list structued data, got %s", s.Tree.Kind())
	}
	s.Source = source
	return nil
}

func NewTerminalUI(ctx context.Context, filePath string) tea.Model {
	state := &cueFileState{}
	window := terminalui.New(
		file.New(filePath),
		// textarea.Textarea{
		// 	Label:    "Enter field",
		// 	Required: true,
		// },
		// list.New(card1, card2),
		// terminalui.NewSwitch(card1, card2, false),
		// card1,
	)

	return terminalui.NewDomainAdaptor(ctx, state.Load, window)
}
