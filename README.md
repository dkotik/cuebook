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

- [ ] Ctrl+J and Ctrl+K entry reordering of entries
- [ ] Ctrl+J and Ctrl+K entry reordering of fields
- [ ] add/delete entry button
- [ ] add/delete field button
- [x] add `@cuebook(title)` attribute support
- [x] add `@cuebook(details)` attribute support
- [ ] add `@cuebook(multiline)` attribute support
- [x] multiple delete and insert cycles leave whitespace artifacts
- [x] add Markdown metadata loading and editing
- [x] add editing history tracking
- [ ] add copy-paste support
- [x] saving should merge the state on disk with the state in memory
- [ ] responsive color scheme: lipgloss.HasDarkBackground() at initialization
- [x] handle change of entry index due to modification of the file by other processes
    - [x] after patch, check if item at current index has the same byte signature as the last change
    - [x] count potential preceding duplicates
    - [x] locate the entry with identical byte content as the last change, taking duplicates into account
- [ ] Add Datastar web user interface server.
- [ ] Add `@cuebook(default=uuid)` attribute support that fills in random IDs for entities that do not have them
- [ ] Add `@cuebook(secret=argon2id)` attribute that hashes and salts input when saved
- [ ] double entry ledger support with `@cuebook(ledger)` attribute
- [ ] turn replace patch into insert patch, if the original entry disappeared, but not without letting the user choose

## Entry Latching

Cuebook assumes that the file may be modified by another process while the user is typing. The original entry may change the location in file or disappear altogether. To apply changes, the program latches on the exact bytes of the entry and looks for them in file before applying the difference. Latching compensates for possible duplicate entries by counting them, and apply the changes to the last one.

Since the process can occasionally apply modification to the wrong entry, in the presence of duplicates, it is best to populate each entry with a unique identifier. To accomplish this, add one of the following tags to an entry field you desire to use for identifier:

- @cuebook(default=UUID,detail)
- @cuebook(default=SFID?node=0&encoding=base58,detail) for shorter [Snow Flake ID](https://en.wikipedia.org/wiki/Snowflake_ID). If a default Cue value is also present (marked with \*), it is used as prefix for the generated identifier. Default Snow Flake node is `0`, and default encoding is `base58`.

When that field is edited and it is empty, the initial value will be populated with generated identifier. `detail` tag conceals the ID from the entry list and display it only when the entry is selected.
