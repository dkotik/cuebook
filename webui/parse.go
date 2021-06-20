package webui

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
)

func parseUUID(r *http.Request) (string, error) {
	p := path.Base(r.URL.Path)
	if len(p) <= 1 {
		return "", fmt.Errorf("UUID value %q rejected", p)
	}
	return p, nil
}

func parseMap(r *http.Request) (v map[string]interface{}, err error) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return
	}
	r.Body.Close()
	v = make(map[string]interface{})
	return v, json.Unmarshal(data, &v)
}
