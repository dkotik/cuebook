/*
Package htmx provides a hyper text interface to the core cuebook package.

It takes a filesystem and presents a file tree of *.cue files. Any one of them
can be viewed or edited via the hyper text interface. The changes are written back to the filesystem through the core package driver.
*/
package htmx

import (
	"io/fs"
	"net/http"
)

func New(fs fs.FS) (http.Handler, error) {

	return nil, nil
}
