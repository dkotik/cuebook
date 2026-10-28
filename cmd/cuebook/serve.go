package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dkotik/cuebook/htmx"
	"github.com/dkotik/cuebook/htmx/agent"
	gitview "github.com/dkotik/cuebook/htmx/git"
	"github.com/urfave/cli/v3"
)

func newServeCommand() *cli.Command {
	return &cli.Command{
		Name:      "serve",
		Usage:     "serve a Cuebook directory over HTTP",
		ArgsUsage: "<directory>",
		Flags: []cli.Flag{
			&cli.IntFlag{Name: "port", Value: 8080, Usage: "localhost port for the server"},
			&cli.BoolFlag{Name: "agent", Usage: "enable the local chat assistant (requires a kronk-tagged build)"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() != 1 {
				return fmt.Errorf("serve requires exactly one directory argument, got %d", cmd.NArg())
			}
			directory := cmd.Args().First()
			var options []htmx.Option
			if cmd.Bool("agent") {
				assistant, err := agent.New(os.DirFS(directory), agent.WithDefaultModel())
				if err != nil {
					return err
				}
				defer assistant.Close(context.Background())
				options = append(options, htmx.WithAgent(assistant))
			}
			return serveDirectory(directory, cmd.Int("port"), options...)
		},
	}
}

func serveDirectory(directory string, port int64, options ...htmx.Option) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("invalid port %d: expected a value between 1 and 65535", port)
	}
	if isGitDirectory(directory) {
		options = append(append([]htmx.Option(nil), options...), serveGitChangesOption(""))
	}
	handler, err := htmx.NewDirectory(directory, options...)
	if err != nil {
		return fmt.Errorf("create Cuebook handler: %w", err)
	}

	address := net.JoinHostPort("127.0.0.1", strconv.FormatInt(port, 10))
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", address, err)
	}
	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
	fmt.Printf("Cuebook serving %s at http://%s/\n", directory, listener.Addr())
	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve Cuebook directory: %w", err)
	}
	return nil
}

func isGitDirectory(directory string) bool {
	info, err := os.Stat(filepath.Join(directory, ".git"))
	return err == nil && info.IsDir()
}

type serveFileLinker struct {
	prefix string
}

func (linker serveFileLinker) GetFileLink(filePath string) (string, error) {
	query := url.Values{}
	query.Set("file", filePath)
	return routeWithPrefix(linker.prefix, "/?"+query.Encode()), nil
}

func serveGitChangesOption(prefix string) htmx.Option {
	return htmx.WithGitChanges(serveFileLinker{prefix: prefix})
}

func routeWithPrefix(prefix, target string) string {
	base := strings.TrimSuffix(prefix, "/")
	if target == "" {
		return base
	}
	return base + "/" + strings.TrimLeft(target, "/")
}

var _ gitview.FileLinker = serveFileLinker{}
