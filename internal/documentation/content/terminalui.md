---
title: Terminal User Interface
description: Cuebook can edit *.cue files from the terminal view.
---

Cuebook can edit *.cue files from the terminal view.

## Shortcuts

- **Ctrl+C** or **Ctrl+\\**: exit, including while the UI is busy. This does not save pending edits.
- **Ctrl+Z**: suspend on Unix; use the shell's `fg` command to resume.
- **Ctrl+L**: clear and redraw the terminal.
- **Ctrl+J** / **Ctrl+K**: move the selected entry down or up in the entry list, or the selected field down or up in the entry view. Fields that are not written in the entry, such as unset schema fields, cannot be moved, and fields and `@cuebook(detail)` fields are reordered separately. Save or discard field edits first.
- **Esc**: go back, or exit from the top-level view.
