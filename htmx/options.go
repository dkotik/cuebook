package htmx

import (
	"fmt"
	"net/http"
	"path"
	"strings"

	gitview "github.com/dkotik/cuebook/htmx/git"
	"github.com/dkotik/htadaptor"
)

type options struct {
	Adaptor        *htadaptor.Adaptor
	ServeMux       *http.ServeMux
	ServeMuxPrefix string
	Agent          AgentHandler
	GitFileLinker  gitview.FileLinker
}

// AgentHandler is implemented by the independent htmx/agent package. Mount
// lets this package provide the route prefix without importing that package.
type AgentHandler interface {
	http.Handler
	Mount(prefix string) http.Handler
}

// Option configures a handler constructor.
type Option func(*options) error

// WithAgent enables an embeddable agent handler under the "agent" route and
// renders its dockable component in full Cuebook pages.
func WithAgent(agent AgentHandler) Option {
	return func(options *options) error {
		if agent == nil {
			return ErrAgentHandlerNil
		}
		if options.Agent != nil {
			return ErrAgentAlreadyConfigured
		}
		options.Agent = agent
		return nil
	}
}

// WithGitChanges enables a Changes view for the source filesystem. The linker
// supplies application URLs for changed paths. source must contain a Git
// repository with a .git directory; status is refreshed whenever the view opens.
func WithGitChanges(linker gitview.FileLinker) Option {
	return func(options *options) error {
		if linker == nil {
			return ErrGitFileLinkerNil
		}
		if options.GitFileLinker != nil {
			return ErrGitChangesAlreadyConfigured
		}
		options.GitFileLinker = linker
		return nil
	}
}

func WithAdaptor(adaptor *htadaptor.Adaptor) Option {
	return func(options *options) error {
		if adaptor == nil {
			return ErrAdaptorNil
		}
		if options.Adaptor != nil {
			return ErrAdaptorAlreadyConfigured
		}
		options.Adaptor = adaptor
		return nil
	}
}

// WithServeMux registers the handler's routes on mux. The returned handler uses
// the supplied mux, allowing the Cuebook routes to share a router with an app.
func WithServeMux(mux *http.ServeMux) Option {
	return func(options *options) error {
		if mux == nil {
			return ErrServeMuxNil
		}
		if options.Adaptor != nil {
			return ErrAdaptorAlreadyConfigured
		}
		options.ServeMux = mux
		return nil
	}
}

// WithServeMuxPrefix mounts the handler's routes beneath prefix. Prefix may be
// supplied with or without a leading slash and one trailing slash; generated
// links and asset URLs use the same prefix.
func WithServeMuxPrefix(prefix string) Option {
	return func(options *options) error {
		if options.ServeMux != nil {
			return ErrServeMuxAlreadyConfigured
		}
		normalized := strings.TrimSpace(prefix)
		if normalized == "" || normalized == "/" {
			options.ServeMuxPrefix = ""
			return nil
		}
		if strings.ContainsAny(normalized, "?#{}\\\\ \t\r\n") {
			return fmt.Errorf("htmx: invalid serve mux prefix %q", normalized)
		}
		if !strings.HasPrefix(normalized, "/") {
			normalized = "/" + normalized
		}
		if strings.HasSuffix(normalized, "/") {
			normalized = strings.TrimSuffix(normalized, "/")
		}
		if normalized == "" || normalized == "/" || path.Clean(normalized) != normalized {
			return fmt.Errorf("htmx: invalid serve mux prefix %q", normalized)
		}
		options.ServeMuxPrefix = normalized
		return nil
	}
}

func routeWithPrefix(prefix, target string) string {
	base := strings.TrimSuffix(prefix, "/")
	if target == "" {
		return base
	}
	return base + "/" + strings.TrimLeft(target, "/")
}

func resolveOptions(opts []Option) (options, error) {
	var configured options
	for index, option := range opts {
		if option == nil {
			return options{}, fmt.Errorf("htmx: option %d is nil", index+1)
		}
		if err := option(&configured); err != nil {
			return options{}, fmt.Errorf("htmx: failed to apply option %d: %w", index+1, err)
		}
	}
	if configured.ServeMux == nil {
		configured.ServeMux = http.NewServeMux()
	}
	if configured.ServeMuxPrefix == "" {
		configured.ServeMuxPrefix = "/"
	} else {
		if !strings.HasPrefix(configured.ServeMuxPrefix, "/") {
			configured.ServeMuxPrefix = "/" + configured.ServeMuxPrefix
		}
		if strings.HasSuffix(configured.ServeMuxPrefix, "/") {
			configured.ServeMuxPrefix = strings.TrimSuffix(configured.ServeMuxPrefix, "/")
		}
	}
	if configured.Adaptor == nil {
		configured.Adaptor = new(htadaptor.New())
	}
	return configured, nil
}
