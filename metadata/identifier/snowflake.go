package identifier

import (
	"net/url"
	"strconv"

	"github.com/bwmarrin/snowflake"
)

func getSnowFlakeNodeNumber(p url.Values) (node int, err error) {
	values, ok := p["node"]
	if !ok {
		return
	}
	total := len(values)
	if total < 1 {
		return
	}
	return strconv.Atoi(values[total-1])
}

// GenerateSnowFlakeID returns a short unique identifier.
// Uses readable, concise, and unambiguous Base58 encoding, which is
// commonly used for Bitcoin addresses, by default.
func GenerateSnowFlakeID(_ string, parameters url.Values) (string, error) {
	number, err := getSnowFlakeNodeNumber(parameters)
	if err != nil {
		return "", err
	}
	encoding, err := GetEncoding(parameters)
	if err != nil {
		return "", err
	}
	// The default settings permit a node number range from 0 to 1023.
	node, err := snowflake.NewNode(int64(number))
	if err != nil {
		return "", err
	}

	switch encoding {
	case EncodingBase64:
		return parameters.Get("prefix") + node.Generate().Base64(), nil
	default:
		return parameters.Get("prefix") + node.Generate().Base58(), nil
	}
}
