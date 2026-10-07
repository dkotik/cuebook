package htmx

import "context"

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

type liveReloadRequest struct{}

func (*liveReloadRequest) Validate(context.Context) error { return nil }

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

type assetRequest struct {
	Name string `schema:"name"`
}

func (*assetRequest) Validate(context.Context) error { return nil }
