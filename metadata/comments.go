package metadata

import (
	"bytes"
	"regexp"
)

var (
	reCommentLine = regexp.MustCompile(`^(\/\/ ?)?([^\S\r\n]*)([^\r\n]*)\r?\n?`)
)

// ReadLeadingComments gathers all comments found at the
// top of the document source. Stops at the first non-empty line
// which is not a comment. Removes comment prefix `//` with
// following space characters from each line. Returns
// the text of the gathered comments and the number
// of bytes where the last one ended in the source.
//
// Treats multiple empty lines as one empty line. Empty lines
// after the last leading comment are ignored.
func ReadLeadingComments(source []byte) (_ []byte, readBytes int) {
	var (
		comments   = &bytes.Buffer{}
		yieldEmpty bool
		matches    [][]byte
	)
	// fmt.Print("-------------------\n")
	for i := 0; i < 1000; i++ {
		matches = reCommentLine.FindSubmatch(source[readBytes:])
		if len(matches[0]) == 0 {
			// fmt.Print("!!!!!\n")
			// fmt.Printf("source: `%s` %v", string(source[readBytes:]), matches)
			break // first match is the entire pattern, if zero, nothing matched
		}
		// fmt.Printf("source: `%s`", string(source[readBytes:]))
		// fmt.Printf(" = [%s]", matches[1])
		// fmt.Printf("[%s]", matches[2])
		// fmt.Printf("[%s]\n", matches[3])

		if len(matches[3]) == 0 {
			yieldEmpty = true
			readBytes += len(matches[0])
			continue
		}
		if len(matches[1]) == 0 {
			break // found a line that is not a comment
		}
		if yieldEmpty {
			_, _ = comments.WriteRune('\n')
			yieldEmpty = false
		}
		_, _ = comments.Write(matches[2])
		_, _ = comments.Write(matches[3])
		_, _ = comments.WriteRune('\n')
		readBytes += len(matches[0])
	}
	return comments.Bytes(), readBytes
}
