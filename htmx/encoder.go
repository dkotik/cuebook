package htmx

import (
	"encoding/json"
	"net/http"

	"github.com/dkotik/htadaptor"
)

type Flasher interface {
	GetFlashMessage() (message string, continueEncoding bool)
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
		message, continueEncoding := flasher.GetFlashMessage()
		if message != "" {
			trigger, err := json.Marshal(map[string]any{
				"cuebook:flash": map[string]string{"message": message},
			})
			if err != nil {
				return err
			}
			w.Header().Set("HX-Trigger", string(trigger))
		}
		if !continueEncoding {
			w.WriteHeader(http.StatusNoContent) // dropped status
			return nil
		}
	}
	if redirecter, ok := response.(Redirecter); ok {
		if location := redirecter.GetRedirect(); location != "" {
			w.Header().Set("HX-Redirect", location)
			if !isHTMX(r) {
				w.Header().Set("Location", location)
				w.WriteHeader(http.StatusSeeOther)
				return nil
			}
		}
	}
	return e.Encoder.Encode(w, r, status, response)
}
