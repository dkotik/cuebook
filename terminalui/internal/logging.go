package internal

import "log/slog"

func (l entryListCards) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Any("total", len(l.Cards)),
		slog.Int("selected", l.SelectedIndex),
	)
}

func (p frontMatterPatch) LogValue() slog.Value {
	return slog.AnyValue(p.Difference())
}
