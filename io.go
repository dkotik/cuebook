package cuebook

import (
	"bytes"
	"errors"
	"io"
	"math"
	"os"
)

type SourceByteRange struct {
	BeginsAt int
	EndsAt   int
}

func (r SourceByteRange) IsValid() bool {
	return r.BeginsAt < r.EndsAt
}

type SourceUpdate struct {
	SourceByteRange
	ReplaceWith []byte
}

func Update(
	w io.Writer,
	filePath string,
	original []byte,
	entry Entry,
	field Field,
) (n int64, err error) {
	source, err := os.ReadFile(filePath)
	if err != nil {
		return 0, err
	}
	book, err := New(source)
	if err != nil {
		return 0, err
	}

	next, err := book.List()
	if err != nil {
		return 0, err
	}

	found := []SourceByteRange{}
	for next.Next() {
		byteRange := GetByteSpanInSource(next.Value())
		if !byteRange.IsValid() {
			continue
		}
		if bytes.Equal(original, source[byteRange.BeginsAt:byteRange.EndsAt]) {
			found = append(found, byteRange)
		}
	}
	if len(found) == 0 {
		return 0, errors.New("entry not found") // TODO: implement error
	}

	return 0, errors.New("not implemented")
}

func getTabulationRecommendationFromTail(b []byte) int {
	spaces := 0
	tabs := 0

loop:
	for i := len(b) - 1; i >= 0; i-- {
		switch b[i] {
		case ' ':
			spaces++
		case '\t':
			tabs++
		default:
			break loop
		}
	}
	if spaces > 1 {
		tabs += int(math.Ceil(float64(spaces) / 4.0))
	}
	return max(tabs, 2)
}
