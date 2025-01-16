# Cue Book

Terminal editor for lists of structured entries in [Cuelang](https://cuelang.org/) format.

Plain text is a resilient and versatile file format. [Cuelang](https://cuelang.org/) makes text files consumable by programs
by including intuitive and flexible structure and validation definitions in the same file as the data being stored. In effect,
it centralizes authority over data consistency in the best possible place, together with the data itself. Consequently, the data
becomes an execution context with contractual enforcement that can be shared reliably by different people and programs.

**Cue Book** empowers the user to edit structured data without having to learn new syntax and to communicate with exact precision
changes in data to other users and, most importantly, programs.

## Use Cases

- Keeping track of transactions or debts.
- Keeping inventory of similar items with their attributes.
- Managing flexible queues of business processes, for which specialized software has not yet been created.
- Controlling services, which do not have an administrative panel yet.
- Editing complex configuration files.
- Delegating any of the above to a secretary.

## Development

- [ ] remove `patch.Nothing` - the difference anchors have to flip
- [ ] add `@cuebook(title)` attribute support
- [ ] add `@cuebook(details)` attribute support
- [ ] add `@cuebook(multiline)` attribute support
- [ ] multiple delete and insert cycles leave whitespace artifacts
- [ ] add Markdown metadata loading and editing
- [x] add editing history tracking
- [ ] add copy-paste support
- [x] saving should merge the state on disk with the state in memory
- [x] Ctrl+J and Ctrl+K entry reordering of entries
- [x] Ctrl+J and Ctrl+K entry reordering of fields
- [ ] responsive color scheme: lipgloss.HasDarkBackground() at initialization
- [x] handle change of entry index due to modification of the file by other processes
    - [x] after patch, check if item at current index has the same byte signature as the last change
    - [x] count potential preceding duplicates
    - [x] locate the entry with identical byte content as the last change, taking duplicates into account
- [ ] Add Datastat web user interface server.
- [ ] Add `@cuebook(uuid)` attribute support that fills in random IDs for entities that do not have them
- [ ] Add `@cuebook(argon2id)` attribute that hashes and salts input when saved
- [ ] double entry ledger support with `@cuebook(ledger)` attribute
