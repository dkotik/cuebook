package metadata

import (
	"cmp"
	"iter"
	"net/url"
	"strings"
	"time"

	"cuelang.org/go/cue"
	"github.com/dkotik/cuebook/metadata/identifier"
)

type AttributeFields iter.Seq[AttributeField]

func (fields AttributeFields) GetFirstOf(key string) (AttributeField, bool) {
	for field := range fields {
		if field.Key == key {
			return field, true
		}
	}
	return AttributeField{}, false
}

func (fields AttributeFields) GetLastOf(key string) (field AttributeField, found bool) {
	for possible := range fields {
		if field.Key == key {
			field = possible
			found = true
		}
		// spew.Dump(field, found)
	}
	return
}

type AttributeField struct {
	Key   string
	Value string
	Query url.Values
}

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
	_, ok = GetFieldAttributes(v, "cuebook").GetFirstOf("title")
	return
}

func IsDetailField(v cue.Value) (ok bool) {
	_, ok = GetFieldAttributes(v, "cuebook").GetFirstOf("detail")
	return
}

func IsMultiLine(v cue.Value) (ok bool) {
	_, ok = GetFieldAttributes(v, "cuebook").GetFirstOf("multiline")
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

	field, ok := GetFieldAttributes(v, "cuebook").GetLastOf("default")
	if !ok {
		return
	}
	call, ok := AttributeDefaults[field.Key]
	if !ok {
		return
	}

	defaultValue, err := call(defaultValue, field.Query)
	if err != nil {
		// panic(err)
		// // TODO: handle error?
		return "", false
	}
	return defaultValue, true
}

func GetFieldAttributes(v cue.Value, name string) AttributeFields {
	return func(yield func(AttributeField) bool) {
		for _, set := range v.Attributes(cue.FieldAttr) {
			if set.Name() != name {
				continue
			}

			var key, value, query string
			var params url.Values
			for i := range set.NumArgs() {
				key, value = set.Arg(i)
				if value == "" {
					key, query, _ = strings.Cut(key, "?")
					if query != "" {
						params, _ = url.ParseQuery(query)
					}
				} else {
					value, query, _ = strings.Cut(value, "?")
					if query != "" {
						params, _ = url.ParseQuery(query)
					}
				}

				if !yield(AttributeField{
					Key:   key,
					Value: value,
					Query: params,
				}) {
					return
				}
				params = nil
			}
		}
	}
}
