package agent

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"
	"time"
)

const defaultModelSource = "unsloth/Qwen3-0.6B-Q8_0"

type options struct {
	model              Generator
	modelSource        string
	cacheDir           string
	serveMuxPrefix     string
	maxFileBytes       int64
	maxIndexedBytes    int64
	maxContextBytes    int
	maxContextFiles    int
	maxMessageBytes    int
	maxOutputTokens    int
	maxHistoryMessages int
	maxSessions        int
	sessionTTL         time.Duration
	requestTimeout     time.Duration
	maxConcurrent      int
}

// Option configures an agent handler.
type Option func(*options) error

// Generator performs one non-streaming chat generation. Implementations must
// honor ctx cancellation. WithModel is useful for tests and custom local models.
type Generator interface {
	Generate(ctx context.Context, messages []Message, maxTokens int) (string, error)
}

// WithModel supplies a generator instead of constructing a Kronk model.
// It cannot be combined with WithModelSource.
func WithModel(model Generator) Option {
	return func(options *options) error {
		if model == nil {
			return errors.New("agent: model is nil")
		}
		if options.model != nil || options.modelSource != "" {
			return errors.New("agent: model is already configured")
		}
		options.model = model
		return nil
	}
}

// WithModelSource configures Kronk to download/cache and load the named local
// model. The model is loaded during New; its weights and Kronk's native runtime
// are stored outside the Go binary in the configured cache directory.
func WithModelSource(source string) Option {
	return func(options *options) error {
		source = strings.TrimSpace(source)
		if source == "" {
			return errors.New("agent: model source is empty")
		}
		if options.model != nil || options.modelSource != "" {
			return errors.New("agent: model is already configured")
		}
		options.modelSource = source
		return nil
	}
}

// WithDefaultModel selects the small Qwen3 0.6B Q8_0 model from the Kronk
// model catalog. First-time construction downloads model weights and native
// inference libraries; subsequent construction reuses the cache.
func WithDefaultModel() Option {
	return WithModelSource(defaultModelSource)
}

// WithCacheDir sets the directory used for Kronk libraries and model weights.
// If omitted, the operating system's user cache directory is used.
func WithCacheDir(directory string) Option {
	return func(options *options) error {
		directory = strings.TrimSpace(directory)
		if directory == "" {
			return errors.New("agent: cache directory is empty")
		}
		options.cacheDir = directory
		return nil
	}
}

// WithServeMuxPrefix mounts this handler's routes beneath prefix. Prefix may
// be supplied with or without a leading slash and one trailing slash.
func WithServeMuxPrefix(prefix string) Option {
	return func(options *options) error {
		normalized, err := normalizePrefix(prefix)
		if err != nil {
			return err
		}
		options.serveMuxPrefix = normalized
		return nil
	}
}

// WithContextLimits configures maximum file size, total indexed content size,
// prompt context size, and the number of files included in one prompt. Values
// must be positive. Files beyond index limits are skipped, not partially read.
func WithContextLimits(maxFileBytes, maxIndexedBytes int64, maxContextBytes, maxFiles int) Option {
	return func(options *options) error {
		if maxFileBytes < 1 || maxFileBytes == int64(^uint64(0)>>1) || maxIndexedBytes < 1 || maxContextBytes < 1 || maxFiles < 1 {
			return errors.New("agent: context limits must be positive")
		}
		if maxFileBytes > maxIndexedBytes || int64(maxContextBytes) > maxIndexedBytes {
			return errors.New("agent: file and prompt limits must not exceed the total index limit")
		}
		options.maxFileBytes = maxFileBytes
		options.maxIndexedBytes = maxIndexedBytes
		options.maxContextBytes = maxContextBytes
		options.maxContextFiles = maxFiles
		return nil
	}
}

// WithSessionLimits sets the maximum number of in-memory sessions, number of
// stored user/assistant messages per session, and idle expiration. Messages are
// truncated to the newest complete turns when the limit is reached.
func WithSessionLimits(maxSessions, maxMessages int, idleTTL time.Duration) Option {
	return func(options *options) error {
		if maxSessions < 1 || maxMessages < 2 || idleTTL <= 0 {
			return errors.New("agent: session limits must be positive and allow at least one turn")
		}
		options.maxSessions = maxSessions
		options.maxHistoryMessages = maxMessages
		options.sessionTTL = idleTTL
		return nil
	}
}

// WithMaxOutputTokens sets the generation limit for each model response.
func WithMaxOutputTokens(limit int) Option {
	return func(options *options) error {
		if limit < 1 || limit > 4096 {
			return errors.New("agent: output token limit must be between 1 and 4096")
		}
		options.maxOutputTokens = limit
		return nil
	}
}

// WithMaxMessageBytes sets the maximum UTF-8 byte length accepted from a
// single user message.
func WithMaxMessageBytes(limit int) Option {
	return func(options *options) error {
		if limit < 1 || limit > 1024*1024 {
			return errors.New("agent: message limit must be between 1 byte and 1 MiB")
		}
		options.maxMessageBytes = limit
		return nil
	}
}

// WithRequestTimeout bounds waiting for inference, including time spent
// waiting for a concurrency slot.
func WithRequestTimeout(timeout time.Duration) Option {
	return func(options *options) error {
		if timeout <= 0 {
			return errors.New("agent: request timeout must be positive")
		}
		options.requestTimeout = timeout
		return nil
	}
}

// WithMaxConcurrentRequests bounds simultaneous model calls.
func WithMaxConcurrentRequests(limit int) Option {
	return func(options *options) error {
		if limit < 1 {
			return errors.New("agent: concurrency limit must be positive")
		}
		options.maxConcurrent = limit
		return nil
	}
}

func defaultOptions() options {
	return options{
		maxFileBytes:       256 * 1024,
		maxIndexedBytes:    8 * 1024 * 1024,
		maxContextBytes:    32 * 1024,
		maxContextFiles:    12,
		maxMessageBytes:    4 * 1024,
		maxOutputTokens:    512,
		maxHistoryMessages: 12,
		maxSessions:        1000,
		sessionTTL:         24 * time.Hour,
		requestTimeout:     2 * time.Minute,
		maxConcurrent:      1,
	}
}

func resolveOptions(opts []Option) (options, error) {
	configured := defaultOptions()
	for index, option := range opts {
		if option == nil {
			return options{}, fmt.Errorf("agent: option %d is nil", index+1)
		}
		if err := option(&configured); err != nil {
			return options{}, fmt.Errorf("agent: failed to apply option %d: %w", index+1, err)
		}
	}
	if configured.model == nil && configured.modelSource == "" {
		return options{}, errors.New("agent: configure a model with WithModel, WithModelSource, or WithDefaultModel")
	}
	return configured, nil
}

func normalizePrefix(prefix string) (string, error) {
	normalized := strings.TrimSpace(prefix)
	if normalized == "" || normalized == "/" {
		return "", nil
	}
	if strings.ContainsAny(normalized, "?#{}\\ \t\r\n") {
		return "", fmt.Errorf("agent: invalid serve mux prefix %q", normalized)
	}
	if !strings.HasPrefix(normalized, "/") {
		normalized = "/" + normalized
	}
	if strings.HasSuffix(normalized, "/") {
		normalized = strings.TrimSuffix(normalized, "/")
	}
	if normalized == "" || path.Clean(normalized) != normalized {
		return "", fmt.Errorf("agent: invalid serve mux prefix %q", normalized)
	}
	return normalized, nil
}
