package line

import (
	"bytes"
	"strings"

	"github.com/charmbracelet/x/ansi/parser"
	"github.com/mattn/go-runewidth"
	"github.com/rivo/uniseg"
)

const (
	bullet   = "•"
	ellipsis = "…"
)

func SplitLines(s string) []string {
	// normalize line endings
	normalized := strings.ReplaceAll(s, "\r\n", "\n")
	return strings.Split(normalized, "\n")
}

func PadLine(s string, w int) string {
	length := runewidth.StringWidth(s)
	if length > w {
		return runewidth.Truncate(s, w-1, ellipsis)
	}
	count := w - length
	if count > 0 {
		b := make([]byte, count)
		for i := range b {
			b[i] = ':'
		}
		return s + string(b)
	}
	return s
}

// TruncateLeft truncates a string from the left side to a given length, adding
// a prefix to the beginning if the string is longer than the given length.
// This function is aware of ANSI escape codes and will not break them, and
// accounts for wide-characters (such as East Asians and emojis).
//
// Copied from: github.com/charmbracelet/x/blob/ansi/v0.6.0/ansi/truncate.go#L113
func TruncateLeft(s string, length int, prefix string) string {
	if length == 0 {
		return ""
	}

	var cluster []byte
	var buf bytes.Buffer
	curWidth := 0
	ignoring := true
	pstate := parser.GroundState
	b := []byte(s)
	i := 0

	for i < len(b) {
		if !ignoring {
			buf.Write(b[i:])
			break
		}

		state, action := parser.Table.Transition(pstate, b[i])
		if state == parser.Utf8State {
			var width int
			cluster, _, width, _ = uniseg.FirstGraphemeCluster(b[i:], -1)

			i += len(cluster)
			curWidth += width

			if curWidth > length && ignoring {
				ignoring = false
				buf.WriteString(prefix)
			}

			if ignoring {
				continue
			}

			if curWidth > length {
				buf.Write(cluster)
			}

			pstate = parser.GroundState
			continue
		}

		switch action {
		case parser.PrintAction:
			curWidth++

			if curWidth > length && ignoring {
				ignoring = false
				buf.WriteString(prefix)
			}

			if ignoring {
				i++
				continue
			}

			fallthrough
		default:
			buf.WriteByte(b[i])
			i++
		}

		pstate = state
		if curWidth > length && ignoring {
			ignoring = false
			buf.WriteString(prefix)
		}
	}

	return buf.String()
}
