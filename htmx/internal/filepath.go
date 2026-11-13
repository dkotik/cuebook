package internal

import (
	"errors"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/dkotik/htadaptor/extract"
)

type filepathExtractor struct {
	routeValue string
	suffix     string
}

// NewFilepathExtractor returns an extractor that converts a ServeMux catch-all path value into the file
// value expected by HTMX request types.
func NewFilepathExtractor(routeValue, suffix string) extract.Extractor {
	return filepathExtractor{routeValue: routeValue, suffix: suffix}
}

func (e filepathExtractor) ExtractStringValue(request *http.Request) (string, error) {
	parts := strings.Split(request.PathValue(e.routeValue), "/")
	joined := path.Join(parts...)
	if joined == "" || joined == "." {
		return "", errors.New("filepath route value is empty")
	}
	return joined + e.suffix, nil
}

func (e filepathExtractor) ExtractRequestValue(values url.Values, request *http.Request) error {
	filePath, err := e.ExtractStringValue(request)
	if err != nil {
		return err
	}
	values.Set(e.routeValue, filePath)
	return nil
}
