package secret

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/matthewhartstonge/argon2"
)

func Argon2ID(input string, parameters url.Values) (string, error) {
	if strings.TrimSpace(input) == "" {
		return "", nil // errors.New("cannot hash an empty password")
	}
	if strings.HasPrefix(input, "$argon2id$") {
		return input, nil
	}
	argon := argon2.DefaultConfig()
	encoded, err := argon.HashEncoded([]byte(input))
	if err != nil {
		return "", fmt.Errorf("unable to hash secret: %w", err)
	}
	return string(encoded), nil
}
