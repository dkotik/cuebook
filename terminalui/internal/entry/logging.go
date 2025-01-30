package entry

import "log/slog"

func (p updateFieldPatch) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Any("entry", p.Entry),
		slog.Any("patch", p.Patch),
	)
}

func (f form) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Any("entry", f.entry),
		slog.Int("fieldCount", len(f.entry.Fields)),
		slog.Int("detailCount", len(f.entry.Details)),
	)
}
