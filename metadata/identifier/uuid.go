package identifier

import (
	"net/url"
	"uuid"
)

func GenerateUUID(_ string, parameters url.Values) (string, error) {
	return parameters.Get("prefix") + uuid.New().String(), nil
}
