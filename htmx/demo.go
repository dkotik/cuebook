package htmx

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// RunDemo serves a writable Cuebook directory on the local IPv4 loopback
// interface. Git repositories get a Changes view linked to their file list;
// opts configure HTMX routes and may enable integrations such as the assistant.
func RunDemo(directory string, port int, opts ...Option) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("invalid port %d: expected a value between 1 and 65535", port)
	}
	if isGitDemoDirectory(directory) {
		opts = append(append([]Option(nil), opts...), demoGitChangesOption())
	}
	handler, err := NewDirectory(directory, opts...)
	if err != nil {
		return fmt.Errorf("create demo handler: %w", err)
	}

	address := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", address, err)
	}
	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
	fmt.Printf("Cuebook demo listening at http://%s/\n", listener.Addr())
	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve demo: %w", err)
	}
	return nil
}

func isGitDemoDirectory(directory string) bool {
	info, err := os.Stat(filepath.Join(directory, ".git"))
	return err == nil && info.IsDir()
}

type demoFileLinker struct {
	prefix string
}

func (linker demoFileLinker) GetFileLink(filePath string) (string, error) {
	return routeWithPrefix(linker.prefix, fileURL(filePath)), nil
}

func demoGitChangesOption() Option {
	return func(configured *options) error {
		if configured.GitFileLinker != nil {
			return nil
		}
		return WithGitChanges(demoFileLinker{prefix: configured.ServeMuxPrefix})(configured)
	}
}
