package main

import (
	"flag"
	"fmt"
	"net/http"
	"time"

	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/webui"
)

func main() {
	flag.Parse()
	var err error
	defer func() {
		if err != nil {
			fmt.Printf("Failed to mount the book: %s.", err.Error())
		}
	}()

	b, err := cuebook.NewBook(flag.Args()...)
	if err != nil {
		return
	}

	s := &http.Server{
		Addr:           "localhost:8080",
		Handler:        webui.CRUDQ(b),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	defer s.Close()
	err = s.ListenAndServe()
}
