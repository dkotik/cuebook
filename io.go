package cuebook

type SourceByteRange struct {
	BeginsAt int
	EndsAt   int
}

func (r SourceByteRange) IsValid() bool {
	return r.BeginsAt < r.EndsAt
}

// func Merge(w io.Writer, source []byte, e Entry) (n int64, err error) {
// 	byteRange, ok := e.GetByteOffsetInSource()
// 	if !ok {
// 		for i := len(source) - 1; i >= 0; i-- {
// 			if source[i] == ']' {
// 				// found end of the list
// 				break
// 			}
// 		}
// 		return 0, errors.New("could not find the end of the entry byte range")
// 	}
// 	if byteRange.BeginsAt > 0 {
// 		n, err = io.Copy(w, bytes.NewReader(source[:byteRange.BeginsAt]))
// 		if err != nil {
// 			return n, err
// 		}
// 	}
// 	more, err := e.WriteTo(w)
// 	n += more
// 	if err != nil {
// 		return n, err
// 	}
// 	if _, err = w.Write([]byte(`,`)); err != nil {
// 		return n, err
// 	}
// 	n++ // for comma
// 	more, err = io.Copy(w, bytes.NewReader(source[byteRange.EndsAt:]))
// 	n += more
// 	if err != nil {
// 		return n, err
// 	}
// 	return n, nil
// }
