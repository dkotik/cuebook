# `htmx/agent`

`htmx/agent` is an opt-in, in-process chat handler for applications using Cuebook's HTMX UI. It snapshots supported files from a caller-provided `fs.FS`, selects bounded context for each question, and invokes a local model through the Kronk Go SDK. The package does not import the parent `htmx` package; the parent accepts it through a small `http.Handler`/`Mount` interface.

## Use with Cuebook's HTMX page

```go
source := os.DirFS("./workspace")

assistant, err := agent.New(source,
    agent.WithDefaultModel(),
)
if err != nil {
    return err
}
defer assistant.Close(context.Background())

app, err := htmx.New(source,
    htmx.WithAgent(assistant),
    htmx.WithServeMuxPrefix("/cuebook"),
)
if err != nil {
    return err
}

// Mount app on the application's server and call assistant.Close during shutdown.
```

The agent is disabled unless `htmx.WithAgent` is provided. When enabled, Cuebook renders the dockable `<cuebook-agent>` custom element and mounts its widget, chat, and embedded assets beneath `/cuebook/agent/` (or `/agent/` at the root). The integration uses HTMX for widget loading and chat submissions. The widget JavaScript handles drag-to-dock, left/right docking, minimize/restore, resizing, and local persistence.

To mount the handler independently, strip the mount path and use `Mount` so generated widget URLs match the route:

```go
mux := http.NewServeMux()
mux.Handle("/assistant/", http.StripPrefix("/assistant", assistant.Mount("/assistant")))
```

An independent host page must load HTMX plus `/assistant/assets/agent.css` and `/assistant/assets/agent.js`, then request `/assistant/widget` into the page with `hx-get`.

## Model setup

`New` requires one of:

- `agent.WithDefaultModel()` — use Kronk's small `unsloth/Qwen3-0.6B-Q8_0` model.
- `agent.WithModelSource(source)` — select another Kronk-supported local model source.
- `agent.WithModel(generator)` — supply a custom `agent.Generator`; useful for tests and other local inference implementations.

Kronk initializes and loads the model synchronously during `New`. Its adapter is enabled with the Go build tag `kronk`; builds without that tag remain usable with `agent.WithModel`, but `WithModelSource` and `WithDefaultModel` return an actionable error instead of loading native FFI code. Build the application with `go build -tags kronk ./...` (and run Kronk-dependent tests with `go test -tags kronk ...`).

A tagged build requires a compatible native `libffi` shared library (including `libffi.8` on platforms where Kronk's FFI dependency requests it) discoverable by the platform loader, in addition to the Kronk native inference libraries. Install/configure those libraries for the target OS and architecture before starting the application. On first model use, Kronk may download model weights and platform-specific native inference libraries. They are cached outside the application binary under the operating system's user cache directory by default; `agent.WithCacheDir` selects a different cache directory. Subsequent startup reuses the cache. Use a cancellable application startup flow or an injected generator if loading should not block server initialization.

The Kronk SDK is Apache-2.0. The default `unsloth/Qwen3-0.6B-Q8_0` GGUF model card currently declares Apache-2.0 for the Qwen3-0.6B model ([model card](https://huggingface.co/unsloth/Qwen3-0.6B-GGUF), [upstream license](https://huggingface.co/Qwen/Qwen3-0.6B/blob/main/LICENSE)); verify the upstream terms before redistributing weights or a pre-populated cache. The model weights are not included in this repository. Kronk's native-library bundles and hardware acceleration vary by operating system and architecture.

## Context and privacy

The context index is a read-only snapshot built at `New`. It supports case-insensitive `.cue`, `.toml`, `.json`, `.yaml`, `.yml`, `.md`, `.markdown`, and `.txt` extensions. Supported files are passed as source text with their relative paths and format labels; malformed structured files are not evaluated or executed. Unsupported files, symlinks, non-regular files, and files above the configured size limits are skipped. Walk/read errors fail construction.

Defaults bound the index to 8 MiB total and 256 KiB per file. Each prompt includes at most 12 files and 32 KiB of selected context. Selection uses deterministic lexical ranking, not embeddings. The current index is not refreshed after construction; create a new handler to pick up filesystem changes.

Inference runs locally in the application process. The agent does not send workspace content to a hosted inference service, but selected file contents and conversation history are included in the local model prompt. First-time Kronk/model setup requires network access to download the model/runtime; that download is separate from chat inference.

Chat history is kept in bounded in-memory sessions identified by an opaque HttpOnly, SameSite cookie. Sessions expire after 24 hours of inactivity, retain up to 12 user/assistant messages, and are lost when the process exits. Requests are same-origin checked, size limited, timeout bounded, and inference concurrency defaults to one request.

## Configuration

Use the functional options to tune resource limits and routing:

```go
assistant, err := agent.New(source,
    agent.WithModelSource("unsloth/Qwen3-0.6B-Q8_0"),
    agent.WithCacheDir("/var/cache/cuebook-agent"),
    agent.WithContextLimits(256*1024, 8*1024*1024, 32*1024, 12),
    agent.WithMaxMessageBytes(4*1024),
    agent.WithMaxOutputTokens(512),
    agent.WithSessionLimits(1000, 12, 24*time.Hour),
    agent.WithRequestTimeout(2*time.Minute),
    agent.WithMaxConcurrentRequests(1),
    agent.WithServeMuxPrefix("/assistant"),
)
```
