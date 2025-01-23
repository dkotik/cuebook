package internal

import (
	"github.com/dkotik/cuebook/terminalui/window"
)

func WithStateEventTransformers() window.Option {
	return window.WithWatchers(
		patchHistoryTracker{},
		flashAnnouncer{},
	)
}
