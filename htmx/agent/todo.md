# Agent package implementation plan and status

## Goal

Implement `github.com/dkotik/cuebook/htmx/agent` as an opt-in, embeddable local chat assistant for the Cuebook HTMX UI. It runs a small model in-process through Kronk, answers using read-only context from a supplied `fs.FS`, and presents chat in a dockable custom HTML element.

**Status:** implementation and focused validation are complete. A native inference smoke test remains environment-dependent because Kronk's FFI runtime requires a system `libffi` shared library that is not installed in the current environment.

## Design decisions and constraints

- Kronk performs in-process inference; prompts and files are not sent to a hosted model service.
- Kronk integration is guarded by the `kronk` Go build tag. This prevents FFI package initialization from breaking ordinary builds and fake-generator tests. Compile production builds with `-tags kronk`; builds without it can use `WithModel` and return an actionable error for Kronk model options.
- The default model is `unsloth/Qwen3-0.6B-Q8_0`. No model weights are committed. See `README.md` for current model license, cache/download behavior, build requirements, and privacy details.
- Kronk-specific code is behind the `Generator` interface. The independent `htmx/agent` package does not import its parent `htmx` package; the parent mounts it through a generic `AgentHandler` interface.
- Context is a construction-time, read-only snapshot. Supported files are treated as text; CUE is never evaluated, and source files are never executed or modified.

## Implementation checklist

### 1. Package API and lifecycle — complete

- [x] Add `agent.New(source fs.FS, opts ...Option)` and an `http.Handler` implementation with a caller-selectable mount/prefix.
- [x] Provide functional options for model source/injection, cache directory, context/session/request/concurrency limits, generation limits, and routing; validate invalid configuration.
- [x] Put the real Kronk generator behind the internal `Generator` interface. Provide a no-Kronk build stub with guidance to rebuild using `-tags kronk`.
- [x] Load/download the model only when an agent is constructed with a Kronk model option; expose idempotent `Close` for owned model resources.
- [x] Pin Kronk in `go.mod`; document that tagged runtime builds need compatible native `libffi` and Kronk runtime libraries.

### 2. Filesystem context — complete

- [x] Index regular, valid relative paths from any supplied `fs.FS`; do not assume `os.DirFS`.
- [x] Support case-insensitive `.cue`, `.toml`, `.json`, `.yaml`, `.yml`, `.md`, `.markdown`, and `.txt` files with path and format labels.
- [x] Preserve source text as data without parsing/evaluating structured formats; skip unsupported files, symlinks, and files over configured limits. Fail construction on walk/read errors.
- [x] Bound per-file, total-index, selected-context, and per-prompt file counts. Rank deterministically by lexical query matches and provide source citations.
- [x] Document that the index is a construction-time snapshot.

### 3. Chat endpoint and sessions — complete

- [x] Implement a form-encoded chat POST endpoint that combines system instructions, selected context, bounded server-side history, and the current user message.
- [x] Use opaque random HttpOnly cookies, bounded in-memory sessions, idle expiry, and bounded history; do not accept client-supplied assistant/system messages.
- [x] Enforce same-origin checks, body/message limits, request deadlines/cancellation, and bounded inference concurrency.
- [x] Return user-safe errors and escape model/user output with `html/template`; show selected source paths.
- [x] Keep chat request/response based; pass request cancellation through to model generation.

### 4. HTMX component and parent integration — complete

- [x] Embed and serve the widget/transcript templates and CSS/JavaScript assets with explicit content types.
- [x] Implement `<cuebook-agent>` with HTMX submission, open/minimize controls, dock selection, pointer dragging, resizing, and persisted position/state.
- [x] Include narrow-screen layout, keyboard-operable controls, labeled input and live transcript, visible focus, and reduced-motion styling.
- [x] Add opt-in `htmx.WithAgent` integration that mounts under the parent's route prefix and conditionally renders the component/assets; no agent import is added to the parent package.

### 5. Tests, documentation, verification — implemented; see validation notes

- [x] Test context format discovery, case-insensitive extensions, unsupported/symlink skipping, file/index/context limits, UTF-8 boundaries, and deterministic selection using `fstest.MapFS`.
- [x] Test chat rendering/context citations/escaping, request validation, session isolation/expiry, model failures, cancellation, and cookie/mount behavior with a fake generator.
- [x] Test embedded asset routes/content types, accessibility-relevant widget markup, and presence of docking/resize/focus/reduced-motion client hooks.
- [x] Add build-tagged parent integration tests for enabled/prefixed and disabled agent rendering.
- [x] Document package construction, mounting, model licensing/setup, context formats/limits, privacy, and native build prerequisites in `README.md`.
- [ ] Run an actual Kronk model inference smoke test on a machine with compatible native `libffi` and model/runtime available; ordinary tests do not download weights.
- [ ] Add browser-driven interaction tests if/when this repository adopts a browser test harness; current component checks verify the served asset contract and markup statically.

## Validation results

Executed with Go caches and temporary test files under `/Users/dima/Library/Caches/genai`:

- `go test ./htmx/agent` — passed.
- `go test -race ./htmx/agent` — passed.
- `go test -tags integration ./htmx -run 'TestHTMX(MountsAgentUnderPrefixedRoute|DoesNotRenderAgentWhenDisabled)$'` — passed.
- `go test -tags kronk -c -o /Users/dima/Library/Caches/genai/agent-kronk.test ./htmx/agent` — passed; validates compilation without executing the native FFI initializer.
- `go vet ./htmx/agent ./htmx` and `git diff --check` — passed.
- `go test ./...` — fails in existing `htmx` rendering assertions (`TestEntryTitleFieldIsOmittedFromEntryContent`, `TestReadOnlyHandler/selected_nested_file_renders_entries`, `TestFileFrontmatterView/frontmatter_details_after_thematic_break_are_collapsible`, and `TestSearchResultsLinkToMatchingEntry`). The failures reproduce with the agent disabled and concern existing parent markup/content expectations; agent changes do not modify those render paths.
- The Kronk-tagged test binary was not run: this environment lacks the required `libffi` shared library (the FFI package panics during initialization). No model weights were downloaded.

## Completion criteria

- [x] Applications can opt into the standalone handler with an `fs.FS` and either a configured Kronk model (build tag required) or injected local `Generator`.
- [x] The optional dockable chat component submits messages through HTMX to the in-process generator and works under parent route prefixes.
- [x] Relevant, bounded, read-only context from all requested file formats reaches the generator with source paths available for citations.
- [x] Core behavior is covered by tests that do not load native libraries or download model weights.
