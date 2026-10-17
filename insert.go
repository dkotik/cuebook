package cuebook

import (
	"bytes"
	"unicode"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/format"
)

type insertAfter struct {
	Preceeding ByteAnchor
	Target     ByteAnchor
}

func (p insertAfter) Difference() ByteAnchor {
	return p.Target
}

func (p insertAfter) ApplyToCueSource(source []byte) ([]byte, error) {
	r, err := p.Preceeding.Match(source)
	if err != nil {
		return nil, err
	}
	return spliceAt(source, r.Tail, p.Target.Content)
}

func (p insertAfter) Invert() Patch {
	return deletePatch(p)
}

func AppendToStructList(source []byte, value cue.Value) (Patch, error) {
	content, err := format.Node(
		value.Syntax(),
		format.Simplify(),
		format.IndentPrefix(1),
		format.UseSpaces(4),
	)
	if err != nil {
		return nil, err
	}

	if len(source) < 2 {
		if len(bytes.TrimSpace(source)) != 0 {
			return nil, ErrSourceIsNotList
		}
		inserted := append([]byte("[\n  "), content...)
		inserted = append(inserted, []byte("\n]\n")...)
		preceding, err := anchorForRange(ByteRange{}, source)
		if err != nil {
			return nil, err
		}
		return insertAfter{
			Preceeding: preceding,
			Target: ByteAnchor{
				Content:              inserted,
				PreceedingDuplicates: bytes.Count(source, inserted),
			},
		}, nil
	}

	compiled := cuecontext.New().CompileBytes(source)
	if compiled.Err() == nil && compiled.Kind() != cue.ListKind {
		return nil, ErrSourceIsNotList
	}

	listEnd, commaFound, empty, err := locateRootListEnd(source)
	if err != nil {
		return nil, err
	}
	if empty || commaFound {
		content = append([]byte("\n  "), content...)
	} else {
		content = append([]byte(",\n  "), content...)
	}
	preceding, err := anchorForRange(ByteRange{
		Head: max(0, listEnd-1000),
		Tail: listEnd,
	}, source)
	if err != nil {
		return nil, err
	}
	return insertAfter{
		Preceeding: preceding,
		Target: ByteAnchor{
			Content:              content,
			PreceedingDuplicates: bytes.Count(source[:listEnd], content),
		},
	}, nil
}

func locateRootListEnd(source []byte) (i int, commaFound, empty bool, err error) {
	if end, ok := emptyTrailingListEnd(source); ok {
		return end, false, true, nil
	}
	listEndFound := false

loop:
	for i := len(source) - 1; i >= max(0, len(source)-1000); i-- {
		switch c := source[i]; c {
		case '}':
			if commaFound {
				i++
			}
			return i + 1, commaFound, false, nil
		case ',':
			if commaFound {
				return 0, commaFound, false, ErrDoubleCommaAtListEnd
			}
			commaFound = true
		case ']':
			if listEndFound {
				return 0, commaFound, false, ErrNotListOfStructs
			}
			listEndFound = true
		default:
			if unicode.IsSpace(rune(c)) {
				continue
			}
			break loop
		}
	}
	return 0, commaFound, false, ErrRootListEndNotFound
}

func emptyTrailingListEnd(source []byte) (int, bool) {
	close := len(source) - 1
	for close >= 0 && unicode.IsSpace(rune(source[close])) {
		close--
	}
	if close < 0 || source[close] != ']' {
		return 0, false
	}

	open := close - 1
	for open >= 0 && unicode.IsSpace(rune(source[open])) {
		open--
	}
	if open >= 0 && source[open] == '[' {
		return close, true
	}
	return 0, false
}
