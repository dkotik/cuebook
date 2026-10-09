package htmx

import (
	"net/http"

	"github.com/dkotik/htadaptor"
)

type Flasher interface {
	GetFlashMessage() string
}

type Redirecter interface {
	GetRedirect() string
}

type encoder struct {
	htadaptor.Encoder
}

func NewEncoder(
	wrap htadaptor.Encoder,
) htadaptor.Encoder {
	return encoder{Encoder: wrap}
}

func (e encoder) Encode(w http.ResponseWriter, r *http.Request, status int, response any) error {
	if flasher, ok := response.(Flasher); ok {
		if message := flasher.GetFlashMessage(); message != "" {
			w.Header().Set("HX-Trigger", message)
		}
	}
	if redirecter, ok := response.(Redirecter); ok {
		if location := redirecter.GetRedirect(); location != "" {
			w.Header().Set("HX-Redirect", location)
			if !isHTMX(r) {
				w.Header().Set("Location", location)
				w.WriteHeader(http.StatusSeeOther)
				return
			}
		}
	}
	return e.Encoder.Encode(w, r, status, response)
}
