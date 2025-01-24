package identifier

import (
	"fmt"
	"net/url"
	"strings"
)

type Encoding uint8

const (
	EncodingUnknown = iota
	EncodingBase58
	EncodingBase64
)

func GetEncoding(p url.Values) (e Encoding, _ error) {
	values, ok := p["encoding"]
	if !ok {
		return EncodingBase58, nil
	}
	total := len(values)
	if total < 1 {
		return EncodingBase58, nil
	}
	switch encoding := strings.ToLower(values[total-1]); encoding {
	case "base64":
		return EncodingBase64, nil
	default:
		return EncodingUnknown, fmt.Errorf("unknown identifier encoding: %q", encoding)
	}
}
