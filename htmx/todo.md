# HTMX package implementation plan

## Goal

Implement `htmx` as an embeddable `net/http` handler that recursively lists `.cue` files from a supplied source, displays their structured entries, and lets users edit values in the source files. Use the existing Cuebook model and patch packages rather than duplicating parsing or source-editing behavior.

## Existing project pieces to build on

- `htmx/htmx.go` declares `New(fs fs.FS) (http.Handler, error)`, but currently returns no handler. Its package comment describes a recursive `.cue` file tree and editing.
- `cuebook.New`, `Document`, `Entry`, and `Field` parse and expose validated CUE lists and their fields.
- `patch.UpdateFieldValue` creates a field-edit patch; `patch.Commit` applies a patch to fresh on-disk bytes, validates the result as a Cuebook document, and replaces the file.
- `htmx/assets/` is empty. `htmx/testdata/` contains root and nested CUE fixtures; use these to exercise recursive file discovery and rendering.

## Design decision to settle first: writable storage

`fs.FS` is read-only, and it does not expose a filesystem path. In contrast, `patch.Commit` requires OS target and swap paths. Therefore the current `New(fs fs.FS)` input cannot, by itself, fulfill the package's promised save behavior for arbitrary `fs.FS` implementations.

Implemented decision: `New(fs.FS)` remains read-only; `NewWithCommitter` accepts an explicit writer, and `NewDirectory` creates a disk-backed handler whose committer delegates to `patch.Commit`. This avoids assuming that an arbitrary `fs.FS` can be written.

## Implementation steps

- [x] Define the handler and storage contracts, including how a missing writer is represented. Keep `New` responsible for validating dependencies and returning a usable handler or an actionable construction error.
- [x] Add a recursive `.cue` discovery layer using `fs.WalkDir`. Normalize and sort relative slash-separated names, skip non-CUE files, and propagate traversal/read errors. Validate requested names against discovered files and reject traversal or non-CUE paths.
- [x] Add HTML templates and local static assets. Render a full page for normal requests and suitable fragments for HTMX requests. Bundle and pin the HTMX client asset locally (including its license) rather than depending on a third-party runtime URL; serve assets from a fixed route.
- [x] Implement GET handlers: an index with the file tree, a selected-file view, and entry/field views or fragments. Parse file bytes with `cuebook.New`; use `Document`/`Entry`/`Field` for display. Surface malformed CUE as an escaped, useful page-level error rather than a panic.
- [x] Implement field-edit POST handling: validate the file, entry index, field name, and submitted value; build a patch with `patch.UpdateFieldValue`; commit through the storage boundary; then render the updated entry/document. Use the patch layer's fresh-source matching and validation to tolerate unrelated external edits, and report missing/changed targets as recoverable conflicts.
- [x] Apply web safety rules throughout: use `html/template` auto-escaping, never mark CUE/user values as trusted HTML, constrain all file access to discovered `.cue` paths, use appropriate status codes and content types, reject unsupported methods, and avoid exposing raw filesystem errors to clients.
- [x] Add focused handler and storage tests with `net/http/httptest` and the existing `htmx/testdata` fixtures. Cover recursive listing, full-page and HTMX-fragment responses, valid edits, invalid CUE/value handling, missing files/entries/fields, path traversal, read-only mode, and commit failures. Use a temporary directory or a test committer for write-path tests.
- [x] Run `gofmt` and the focused `go test ./htmx/...`; then run `go test ./...` to check integration with the existing packages.

## Completion criteria

- `New` returns a functioning handler with documented read-only and writable-source behavior.
- A browser can navigate the recursive CUE file tree, inspect validated entries, and submit edits that are committed through the existing patch workflow when a writer is configured.
- Malformed input, invalid paths, and write conflicts produce safe, understandable HTTP responses without panics or unintended file access.
- Tests cover the route behavior and verify that successful edits change the stored CUE source while preserving valid CUE structure.
