package markdown

import (
	"fmt"
	"os"
	"sync"

	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	"charm.land/glamour/v2/ansi"
	"charm.land/glamour/v2/styles"
	"charm.land/lipgloss/v2"
	"github.com/dkotik/cuebook/terminalui/window"
)

// Configuration controls Markdown content and appearance.
//
// Style names a glamour standard style, such as "dark", "light", or "dracula".
// When empty, it defaults to "dark" or "light" to match the terminal background.
// StyleConfig, when set, replaces the named style entirely.
type Configuration struct {
	Text        string
	Style       string
	StyleConfig *ansi.StyleConfig
}

// The background query is slow, so run it once and only when needed.
var backgroundStyle = sync.OnceValue(func() string {
	if lipgloss.HasDarkBackground(os.Stdin, os.Stdout) {
		return styles.DarkStyle
	}
	return styles.LightStyle
})

func New(c Configuration) (_ tea.Model, err error) {
	if c.StyleConfig == nil {
		if c.Style == "" {
			c.Style = backgroundStyle()
		}
		config, ok := styles.DefaultStyles[c.Style]
		if !ok {
			return nil, fmt.Errorf("unknown Markdown style %q", c.Style)
		}
		c.StyleConfig = config
	}
	return markdown{width: 40, text: c.Text, style: *c.StyleConfig}, nil
}

type markdown struct {
	width int
	text  string
	style ansi.StyleConfig
}

func (m markdown) Init() tea.Cmd {
	return window.RequestWindowSize()
}

func (m markdown) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = max(10, msg.Width-2)
		// TODO: loading screen
	}
	return m, nil
}

func (m markdown) View() tea.View {
	r, err := glamour.NewTermRenderer(
		glamour.WithWordWrap(m.width),
		glamour.WithStyles(m.style),
	)
	if err != nil {
		return tea.NewView(m.text)
	}
	out, err := r.Render(m.text)
	if err != nil {
		return tea.NewView(m.text)
	}
	return tea.NewView(out)
}
