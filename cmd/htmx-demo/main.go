package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/dkotik/cuebook/htmx"
	"github.com/dkotik/cuebook/htmx/agent"
)

func main() {
	port := flag.Int("port", 8080, "localhost port for the demo server")
	enableAgent := flag.Bool("agent", false, "enable the local chat assistant (requires a kronk-tagged build)")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "unexpected positional arguments")
		os.Exit(2)
	}
	var options []htmx.Option
	if *enableAgent {
		assistant, err := agent.New(os.DirFS("htmx/testdata"), agent.WithDefaultModel())
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		defer assistant.Close(context.Background())
		options = append(options, htmx.WithAgent(assistant))
	}
	if err := htmx.RunDemo("htmx/testdata", *port, options...); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
