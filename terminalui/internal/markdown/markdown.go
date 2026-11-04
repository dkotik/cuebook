package markdown

import (
	"fmt"
	"sync"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
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
	if lipgloss.HasDarkBackground() {
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
	return markdown{text: c.Text, style: *c.StyleConfig}, nil
}

type markdown struct {
	width int
	text  string
	style ansi.StyleConfig
}

func (m markdown) Init() (tea.Model, tea.Cmd) {
	m.width = 40
	return m, tea.RequestWindowSize()
}

func (m markdown) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = max(10, msg.Width-2)
		// TODO: loading screen
	}
	return m, nil
}

func (m markdown) View() string {
	r, err := glamour.NewTermRenderer(
		glamour.WithWordWrap(m.width),
		glamour.WithStyles(m.style),
	)
	if err != nil {
		return m.text
	}
	out, err := r.Render(m.text)
	if err != nil {
		return m.text
	}
	return out
}
