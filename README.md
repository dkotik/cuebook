# Cue Book

Manage lists of structured data stored in [Cuelang](https://cuelang.org/) files.

## Development

- [ ] add `@cuebook(details)` attribute support
- [ ] add `@cuebook(multiline)` attribute support
- [ ] add `@cuebook(deleteTo=file)` attribute support
- [ ] add Markdown metadata loading and editing
- [ ] Ctrl+up and down entry reordering inside lists
- [ ] responsive color scheme: lipgloss.HasDarkBackground() at initialization
- [x] handle change of entry index due to modification of the file by other processes
    - [x] after patch, check if item at current index has the same byte signature as the last change
    - [x] count potential preceding duplicates
    - [x] locate the entry with identical byte content as the last change, taking duplicates into account
- [ ] double entry ledger support with `@cuebook(ledger)` attribute
