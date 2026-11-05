package middleware

import tea "charm.land/bubbletea/v2"

type Middleware interface {
	Wrap(tea.Model) tea.Model
}

type MiddlewareFunc func(tea.Model) tea.Model

func (mw MiddlewareFunc) Wrap(m tea.Model) tea.Model {
	return mw(m)
}
