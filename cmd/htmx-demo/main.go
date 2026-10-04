//go:build demo

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/dkotik/cuebook/htmx"
)

func main() {
	port := flag.Int("port", 8080, "localhost port for the demo server")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "unexpected positional arguments")
		os.Exit(2)
	}
	if err := htmx.RunDemo("htmx/testdata", *port); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
