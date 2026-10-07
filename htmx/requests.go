package htmx

import (
	"context"
	"fmt"
	"net/http"
)

type listRequest struct {
	File string `schema:"file"`
}

func (*listRequest) Validate(context.Context) error { return nil }

type editFormRequest struct {
	File  string `schema:"file"`
	Entry string `schema:"entry"`
	Field string `schema:"field"`
	Mode  string `schema:"mode"`
}

func (*editFormRequest) Validate(context.Context) error { return nil }

type itemRouteRequest struct {
	Path string `schema:"path"`
	File string `schema:"file"`
	Head string `schema:"head"`
	Tail string `schema:"tail"`
}

func (*itemRouteRequest) Validate(context.Context) error { return nil }

type searchRequest struct {
	Query string `schema:"q"`
	Alt   string `schema:"query"`
}

func (*searchRequest) Validate(context.Context) error { return nil }

type searchClearRequest struct {
	Query string `schema:"q"`
}

func (*searchClearRequest) Validate(context.Context) error { return nil }

type editRequest struct {
	File   string   `schema:"file"`
	Entry  string   `schema:"entry"`
	Field  string   `schema:"field"`
	Values []string `schema:"value"`
}

func (*editRequest) Validate(context.Context) error { return nil }

type addRequest struct {
	File   string   `schema:"file"`
	Fields []string `schema:"field"`
	Values []string `schema:"value"`
}

func (*addRequest) Validate(context.Context) error { return nil }

type moveRequest struct {
	File        string `schema:"file"`
	From        string `schema:"from"`
	To          string `schema:"to"`
	Destination string `schema:"destination"`
}

func (*moveRequest) Validate(context.Context) error { return nil }

type archiveRequest struct {
	File  string `schema:"file"`
	Entry string `schema:"entry"`
}

func (*archiveRequest) Validate(context.Context) error { return nil }

const maxFormBodySize = 1 << 20

type formDecoder struct{}

func (formDecoder) Decode(target any, request *http.Request) error {
	if request.Body != nil {
		request.Body = http.MaxBytesReader(nil, request.Body, maxFormBodySize)
	}
	if err := request.ParseForm(); err != nil {
		return err
	}

	values := request.PostForm
	switch request := target.(type) {
	case *editRequest:
		request.File = values.Get("file")
		request.Entry = values.Get("entry")
		request.Field = values.Get("field")
		request.Values = append([]string(nil), values["value"]...)
	case *addRequest:
		request.File = values.Get("file")
		request.Fields = append([]string(nil), values["field"]...)
		request.Values = append([]string(nil), values["value"]...)
	case *moveRequest:
		request.File = values.Get("file")
		request.From = values.Get("from")
		request.To = values.Get("to")
		request.Destination = values.Get("destination")
	case *archiveRequest:
		request.File = values.Get("file")
		request.Entry = values.Get("entry")
	default:
		return fmt.Errorf("unsupported form request %T", target)
	}
	return nil
}
