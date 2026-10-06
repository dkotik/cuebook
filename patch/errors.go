package patch

import (
	"fmt"

	"github.com/dkotik/cuebook"
)

type Error uint8

const (
	ErrUnknown Error = iota
	ErrSourceIsNotList
	_
	ErrByteRangesOverlap
)

// ErrByteRangeNotFound aliases the root package's byte-range lookup error.
var ErrByteRangeNotFound = cuebook.ErrByteRangeNotFound

func (e Error) Error() string {
	switch e {
	// TODO: fill out
	default:
		return fmt.Sprintf("unknown patch error code #%d", e)
	}
}
