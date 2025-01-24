package identifier

import (
	"net/url"

	"github.com/bwmarrin/snowflake"
)

func GenerateSnowFlakeID(input string, parameters url.Values) (string, error) {
	node, err := snowflake.NewNode(0) // TODO: parameterize
	if err != nil {
		return "", err
	}
	return input + node.Generate().Base58(), nil
}
