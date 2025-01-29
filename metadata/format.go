package metadata

import (
	"fmt"
	"net/url"
	"slices"
	"strings"

	"cuelang.org/go/cue"
	"github.com/dkotik/cuebook/metadata/secret"
)

var formatTransformers = map[string]FieldTransformer{
	"trim": func(input string, parameters url.Values) (string, error) {
		return strings.TrimSpace(input), nil
	},
	"argon2id": secret.Argon2ID,
}

func FormatAccordingToAttributes(v cue.Value, input string) (string, error) {
	for _, attribute := range v.Attributes(cue.FieldAttr) {
		if attribute.Name() == "cuebook" {
			for i := range attribute.NumArgs() {
				format, params := attribute.Arg(i)
				call, ok := formatTransformers[format]
				if !ok {
					if slices.Index([]string{
						"title",
						"detail",
					}, format) > -1 {
						continue
					}
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
