package identifier

import (
	"net/url"

	"github.com/google/uuid"
)

func GenerateUUID(_ string, parameters url.Values) (string, error) {
	UUID, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	return parameters.Get("prefix") + UUID.String(), nil
}
