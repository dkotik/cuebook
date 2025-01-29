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

func FormatAccordingToAttributes(v cue.Value, input string) (_ string, err error) {
	for attributeField := range GetFieldAttributes(v, "cuebook") {
		call, ok := formatTransformers[attributeField.Key]
		if !ok {
			if attributeField.Key == "default" {
				if input == "" { // TODO: move this logic into UI form to populate the form, rather than apply here
					defaultCall, ok := AttributeDefaults[attributeField.Value]
					if !ok {
						return "", fmt.Errorf("no such default function: %s", attributeField.Value)
					}
					input, err = defaultCall("", attributeField.Query)
					if err != nil {
						return "", fmt.Errorf("unable to apply default value: %w", err)
					}
				}
				continue
			}
			if slices.Index([]string{
				"title",
				"detail",
				"multiline",
			}, attributeField.Key) > -1 {
				continue
			}
			return "", fmt.Errorf("format function does not exist: %s", attributeField.Key)
		}
		input, err = call(input, attributeField.Query)
		if err != nil {
			return "", fmt.Errorf("unable to execute format function %q: %w", attributeField.Key, err)
		}
	}
	return input, nil
}
