package metadata

import (
	"fmt"
	"net/url"
	"strings"

	"cuelang.org/go/cue"
	"github.com/matthewhartstonge/argon2"
)

var formatTransformers = map[string]FieldTransformer{
	"trim": func(input string, parameters url.Values) (string, error) {
		return strings.TrimSpace(input), nil
	},
	"argon2id": func(input string, parameters url.Values) (string, error) {
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
	},
}

func FormatAccordingToAttributes(v cue.Value, input string) (string, error) {
	for _, attribute := range v.Attributes(cue.FieldAttr) {
		if attribute.Name() == "cuebook" {
			for i := range attribute.NumArgs() {
				format, params := attribute.Arg(i)
				call, ok := formatTransformers[format]
				if !ok {
					return "", fmt.Errorf("format function does not exist: %s", format)
				}
				parsedParams, err := url.ParseQuery(params)
				if err != nil {
					return "", fmt.Errorf("cannot parse format function parameters: %w", err)
				}
				input, err = call(input, parsedParams)
				if err != nil {
					return "", fmt.Errorf("unable to execute format function %q: %w", format, err)
				}
			}
		}
	}
	return input, nil
}
