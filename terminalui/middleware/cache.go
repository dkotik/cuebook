package middleware

import (
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea/v2"
)

var cacheID atomic.Uint64

type cachedView struct {
	CacheID uint64
	View    string
	Time    int64
}

func NewCache(preview string) Middleware {
	return MiddlewareFunc(func(model tea.Model) tea.Model {
		if model == nil {
			panic("model is nil")
		}
		return cachedModel{
			Model:      model,
			view:       preview,
			debounce:   make(chan struct{}, 1),
			lastUpdate: time.Now().UnixNano(),
		}
	})
}

type cachedModel struct {
	tea.Model

	debounce   chan struct{}
	ID         uint64
	view       string
	lastUpdate int64
}

func (c cachedModel) Init() (model tea.Model, cmd tea.Cmd) {
	c.Model, cmd = c.Model.Init()
	c.ID = cacheID.Add(1)
	return c, tea.Batch(cmd, c.Render())
}

func (c cachedModel) Render() tea.Cmd {
	select {
	case c.debounce <- struct{}{}:
		return func() tea.Msg {
			view := c.Model.View()
			// time.Sleep(time.Second)
			<-c.debounce
			return cachedView{
				CacheID: c.ID,
				View:    view,
				Time:    c.lastUpdate,
			}
		}
	default:
		return nil
	}
}

func (c cachedModel) Update(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	switch msg := msg.(type) {
	case cachedView:
		if msg.CacheID == c.ID {
			c.view = msg.View
			if c.lastUpdate > msg.Time {
				return c, c.Render()
			}
			return c, nil
		}
	}
	c.Model, cmd = c.Model.Update(msg)
	c.lastUpdate = time.Now().UnixNano()
	// TODO: contribute tea.Cmd debounce to bubbletea
	return c, tea.Batch(cmd, c.Render())
}

func (c cachedModel) View() string {
	return c.view
}
