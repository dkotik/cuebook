//go:build demo

package htmx

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"
)

// RunDemo serves a writable Cuebook directory on the local IPv4 loopback
// interface. It is available only in builds that include the demo tag.
func RunDemo(directory string, port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("invalid port %d: expected a value between 1 and 65535", port)
	}
	handler, err := NewDirectory(directory)
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
