package internal

import "log/slog"

func (l entryListCards) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Any("total", len(l.Cards)),
		slog.Int("selected", l.SelectedIndex),
	)
}

func (p updateFieldPatch) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Any("entry", p.Entry),
		slog.Any("patch", p.Patch),
	)
}

func (p frontMatterPatch) LogValue() slog.Value {
	return slog.AnyValue(p.Difference())
}

func (l FieldList) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Any("entry", l.entry),
		slog.Int("fieldCount", len(l.entry.Fields)),
		slog.Int("detailCount", len(l.entry.Details)),
	)
}
