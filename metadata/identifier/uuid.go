package identifier

import (
	"net/url"

	"github.com/google/uuid"
)

func GenerateUUID(input string, parameters url.Values) (string, error) {
	UUID, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	return input + UUID.String(), nil
}
