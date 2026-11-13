package internal

import (
	"bytes"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/dkotik/cuebook"
	goldmarkmarkdown "github.com/dkotik/cuebook/internal/markdown"
	"github.com/dkotik/cuebook/metadata"
	"github.com/dkotik/cuebook/terminalui/internal/list"
	"github.com/dkotik/cuebook/terminalui/internal/markdown"
	"github.com/dkotik/cuebook/terminalui/internal/textarea"
	"github.com/dkotik/cuebook/terminalui/window"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

type (
	frontMatterListItems []tea.Model
	frontMatterUpdate    struct{}

	frontMatterPatch struct {
		cuebook.Patch
	}
)

func (p frontMatterPatch) UnwrapPatch() cuebook.Patch { return p.Patch }

type FrontMatterView struct {
	tea.Model

	state cuebook.Result
}

func newFrontMatterView() tea.Model {
	return FrontMatterView{Model: list.New(list.Configuration{})}
}

func (v FrontMatterView) Init() tea.Cmd {
	return v.Model.Init()
}

func (v FrontMatterView) Update(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	switch msg := msg.(type) {
	case cuebook.Result:
		if v.state.IsEqual(msg) {
			return v, nil
		}
		v.state = msg
		if _, ok := msg.LastChange.(frontMatterPatch); ok {
			cmd = func() tea.Msg { return window.BackEvent{} }
		}

		return v, tea.Batch(
			cmd, // nil or flip back if recognized as frontMatterPatch
			func() tea.Msg {
				return window.TranslatableFunc(func(lc *i18n.Localizer) tea.Cmd {
					return func() tea.Msg {
						md := metadata.NewFrontmatter(msg.Source, goldmarkmarkdown.NewParser())
						description, err := markdown.New(markdown.Configuration{Text: string(md.Source)})
						if err != nil {
							return err
						}
						return frontMatterListItems{
							description,
							list.NewButton(&i18n.LocalizeConfig{
								DefaultMessage: &i18n.Message{
									ID:    "bookMetadataUpdateDescription",
									Other: "Update description",
								},
							}, func() tea.Msg { return frontMatterUpdate{} }),
						}
					}
				})
			},
		)
	case frontMatterListItems:
		v.Model, cmd = v.Model.Update(list.SetItems(msg...)())
		return v, cmd
	case frontMatterUpdate:
		source := v.state.Source
		return v, func() tea.Msg {
			md := metadata.NewFrontmatter(source, goldmarkmarkdown.NewParser())
			form, err := textarea.New(textarea.Configuration{
				Label: "Description",
				Value: string(md.Source),
				OnSubmit: func(value string) tea.Cmd {
					return func() tea.Msg {
						b := &bytes.Buffer{}
						for _, line := range strings.Split(strings.TrimSpace(value), "\n") {
							_, _ = b.Write([]byte(`// `))
							_, _ = b.WriteString(line)
							_, _ = b.WriteRune('\n')
						}
						_, _ = b.WriteRune('\n') // skip a line after comments
						p, err := cuebook.UpdateRange((cuebook.ByteRange{
							Head: 0,
							Tail: md.TailBytePosition,
						}).Anchor(source), b.Bytes())
						if err != nil {
							return err
						}
						return frontMatterPatch{Patch: p}
					}
				},
			})
			if err != nil {
				return err
			}
			return window.SwitchTo(form)
		}
	}
	v.Model, cmd = v.Model.Update(msg)
	return v, cmd
}
