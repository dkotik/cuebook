package webui

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/dkotik/cuebook"
)

// CRUDQ provides a create, retrieve, update, and delete HTTP interfaces with a search and list Query capability.
func CRUDQ(b *cuebook.Book) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		defer func() {
			switch err {
			case nil:
				return
			case io.EOF:
				http.Error(w, "Not Found", http.StatusNotFound)
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}()

		switch r.Method {
		case http.MethodOptions:
			w.Header().Add("Access-Control-Allow-Methods", "GET,PUT,POST,DELETE,OPTIONS")
			w.WriteHeader(http.StatusOK)
			return
		case http.MethodPost:
			m, err := parseMap(r)
			if err != nil {
				return
			}
			uuid, err := b.Create(r.URL.Path, m)
			if err != nil {
				return
			}
			http.Redirect(w, r, uuid, http.StatusTemporaryRedirect)
			return
		case http.MethodPut:
			m, err := parseMap(r)
			if err != nil {
				return
			}
			uuid, err := parseUUID(r)
			if err != nil {
				return
			}
			err = b.Update(uuid, m)
			return
		case http.MethodDelete:
			uuid, err := parseUUID(r)
			if err != nil {
				return
			}
			err = b.Delete(uuid)
			return
		}

		// GET results
		uuid, err := parseUUID(r)
		if err != nil {
			return
		}
		m, err := b.Retrieve(uuid)
		if err != nil {
			return
		}
		b, err := json.Marshal(m)
		if err != nil {
			return
		}
		w.Write(b)
	}
}
