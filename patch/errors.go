package patch

import "fmt"

type Error uint8

const (
	ErrUnknown Error = iota
	ErrSourceIsNotList
	ErrByteRangeNotFound
	ErrByteRangesOverlap
)

func (e Error) Error() string {
	switch e {
	// TODO: fill out
	default:
		return fmt.Sprintf("unknown patch error code #%d", e)
	}
}
