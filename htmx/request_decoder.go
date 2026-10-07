package htmx

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"reflect"
	"strings"
)

const maxRequestBodySize = 1 << 20

// requestDecoder preserves empty values in repeated form fields, which add-entry
// forms use to keep field names and values aligned.
type requestDecoder struct{}

func (requestDecoder) Decode(target any, request *http.Request) error {
	values := make(url.Values)
	if request.Method == http.MethodGet {
		parsed, err := url.ParseQuery(request.URL.RawQuery)
		if err != nil {
			return err
		}
		values = parsed
	} else if request.Body != nil {
		body, err := io.ReadAll(io.LimitReader(request.Body, maxRequestBodySize+1))
		closeErr := request.Body.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
		if len(body) > maxRequestBodySize {
			return fmt.Errorf("request body exceeds %d bytes", maxRequestBodySize)
		}
		contentType := strings.TrimSpace(request.Header.Get("Content-Type"))
		if contentType != "" {
			mediaType, parameters, err := mime.ParseMediaType(contentType)
			if err != nil {
				return err
			}
			switch mediaType {
			case "application/x-www-form-urlencoded":
				values, err = url.ParseQuery(string(body))
				if err != nil {
					return err
				}
			case "multipart/form-data":
				boundary := parameters["boundary"]
				if boundary == "" {
					return http.ErrMissingBoundary
				}
				form, err := multipart.NewReader(bytes.NewReader(body), boundary).ReadForm(maxRequestBodySize)
				if err != nil {
					return err
				}
				values = form.Value
				if err := form.RemoveAll(); err != nil {
					return err
				}
			default:
				return fmt.Errorf("unsupported request content type %q", mediaType)
			}
		}
	}
	if pathName := request.PathValue("name"); pathName != "" {
		values.Set("name", pathName)
	}

	value := reflect.ValueOf(target)
	if value.Kind() != reflect.Pointer || value.IsNil() || value.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("request target must be a non-nil pointer to struct")
	}
	value = value.Elem()
	typeOfValue := value.Type()
	for index := 0; index < value.NumField(); index++ {
		field := value.Field(index)
		fieldType := typeOfValue.Field(index)
		if !field.CanSet() {
			continue
		}
		name := fieldType.Tag.Get("schema")
		if name == "" || name == "-" {
			continue
		}
		inputs, exists := values[name]
		if !exists {
			continue
		}
		switch field.Kind() {
		case reflect.String:
			if len(inputs) > 0 {
				field.SetString(inputs[0])
			}
		case reflect.Slice:
			if field.Type().Elem().Kind() != reflect.String {
				return fmt.Errorf("unsupported request field %q of type %s", fieldType.Name, field.Type())
			}
			field.Set(reflect.ValueOf(append([]string(nil), inputs...)))
		default:
			return fmt.Errorf("unsupported request field %q of type %s", fieldType.Name, field.Type())
		}
	}
	return nil
}
