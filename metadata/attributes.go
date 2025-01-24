package metadata

import (
	"cmp"
	"net/url"
	"strings"
	"time"

	"cuelang.org/go/cue"
	"github.com/dkotik/cuebook/metadata/identifier"
)

type FieldTransformer func(string, url.Values) (string, error)

var AttributeDefaults = map[string]FieldTransformer{
	"date": func(input string, parameters url.Values) (string, error) {
		var t time.Time
		if input == "" {
			t = time.Now()
		} else if input == "yesterday" || parameters.Has("yesterday") {
			t = time.Now().Add(time.Hour * -24)
		} else if input == "tomorrow" || parameters.Has("tomorrow") {
			t = time.Now().Add(time.Hour * 24)
		} else {
			var err error
			t, err = time.Parse(time.DateOnly, input)
			if err != nil {
				return "", err
			}
		}
		return t.Format(cmp.Or(parameters.Get("format"), time.DateOnly)), nil
	},
	"UUID": identifier.GenerateUUID,
	"SFID": identifier.GenerateSnowFlakeID,
}

func IsTitleField(v cue.Value) (ok bool) {
	_, ok = GetAttribute(v, "cuebook", "title")
	return
}

func IsDetailField(v cue.Value) (ok bool) {
	_, ok = GetAttribute(v, "cuebook", "detail")
	return
}

func IsMultiLine(v cue.Value) (ok bool) {
	_, ok = GetAttribute(v, "cuebook", "multiline")
	return
}

func GetDefaultValue(v cue.Value) (defaultValue string, ok bool) {
	_, exprs := v.Expr()
	for _, expr := range exprs {
		expressionValue, ok := expr.Default()
		if ok {
			defaultValue = strings.TrimSpace(ValueToString(expressionValue))
			break
		}
	}

	functionWithParameters, ok := GetAttribute(v, "cuebook", "default")
	if !ok {
		return
	}
	function, parameterBatch, _ := strings.Cut(functionWithParameters, "?")
	if function == "" {
		return
	}
	call, ok := AttributeDefaults[function]
	if !ok {
		return
	}
	parameters, _ := url.ParseQuery(parameterBatch)
	// if err != nil {
	// 	// TODO: handle error
	// 	return
	// }

	defaultValue, err := call(defaultValue, parameters)
	if err != nil {
		// panic(err)
		// // TODO: handle error?
		return "", false
	}
	return defaultValue, true
}

// GetAttribute return the last value of the attribute with the given name and key.
// If the attribute definition is `@name(key="value1",key="value2")`, the function will
// return `value2`. If there are two keys found for the same name, the latter overrides the former.
func GetAttribute(v cue.Value, name string, key string) (value string, found bool) {
	var (
		possibleKey, possibleValue string
	)
	for _, attribute := range v.Attributes(cue.FieldAttr) {
		if attribute.Name() == name {
			for i := range attribute.NumArgs() {
				possibleKey, possibleValue = attribute.Arg(i)
				if possibleKey == key {
					found = true
					value = possibleValue
				}
			}
		}
	}
	return value, found
}
