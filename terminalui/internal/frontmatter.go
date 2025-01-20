package internal

import (
	"bytes"
	"strings"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/dkotik/cuebook/metadata"
	"github.com/dkotik/cuebook/patch"
	"github.com/dkotik/cuebook/terminalui/list"
	"github.com/dkotik/cuebook/terminalui/markdown"
	"github.com/dkotik/cuebook/terminalui/textarea"
	"github.com/dkotik/cuebook/terminalui/window"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

type (
	frontMatterListItems []tea.Model
	frontMatterUpdate    struct{}
)

type FrontMatterView struct {
	tea.Model

	state patch.Result
}

func (v FrontMatterView) Init() (_ tea.Model, cmd tea.Cmd) {
	v.Model = list.New()
	return v, nil
}

func (v FrontMatterView) Update(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	switch msg := msg.(type) {
	case patch.Result:
		if v.state.IsEqual(msg) {
			return v, nil
		}

		v.state = msg
		return v, func() tea.Msg {
			md := metadata.NewFrontmatter(msg.Source)
			return frontMatterListItems{
				markdown.New(string(md.Source)),
				list.NewButton(&i18n.LocalizeConfig{
					DefaultMessage: &i18n.Message{
						ID:    "bookMetadataUpdateDescription",
						Other: "Update description",
					},
				}, func() tea.Msg { return frontMatterUpdate{} }),
			}
		}
	case frontMatterListItems:
		v.Model, cmd = v.Model.Update(list.SetItems(msg...)())
		return v, cmd
	case frontMatterUpdate:
		source := v.state.Source
		return v, func() tea.Msg {
			md := metadata.NewFrontmatter(source)
			formWrapper, patchWrapper := NewPatchCloser("metadataPatch")
			form, err := textarea.New(
				textarea.WithLabel("Description"),
				textarea.WithValue(string(md.Source)),
				textarea.WithOnSubmitCommand(func(value string) tea.Cmd {
					return func() tea.Msg {
						b := &bytes.Buffer{}
						for _, line := range strings.Split(strings.TrimSpace(value), "\n") {
							_, _ = b.Write([]byte(`// `))
							_, _ = b.WriteString(line)
							_, _ = b.WriteRune('\n')
						}
						_, _ = b.WriteRune('\n') // skip a line after comments
						p, err := patch.UpdateRange(source, md.ByteRange, b.Bytes())
						if err != nil {
							return err
						}
						return patchWrapper(p)
					}
				}),
			)
			if err != nil {
				return err
			}
			return window.SwitchTo(formWrapper(form))
		}
	}
	v.Model, cmd = v.Model.Update(msg)
	return v, cmd
}
