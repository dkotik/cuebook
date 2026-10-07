package htmx

import (
	"context"
	"net/http"
)

func (a *handler) asset(_ context.Context, request *assetRequest) (assetResponse, error) {
	var contentType string
	switch request.Name {
	case "app.css", "bulma.css":
		contentType = "text/css; charset=utf-8"
	case "theme.js", "file-tree.js", "entry-move.js", "remember-details.js", "delete-confirm.js", "move-confirm.js", "live-reload.js":
		contentType = "text/javascript; charset=utf-8"
	case "bulma-LICENSE.txt", "htmx-LICENSE.txt":
		contentType = "text/plain; charset=utf-8"
	case "htmx-2.0.4.min.js":
		contentType = "text/javascript; charset=utf-8"
	case "favicon.svg":
		contentType = "image/svg+xml"
	default:
		return assetResponse{statusCode: http.StatusNotFound, message: "404 page not found"}, nil
	}
	content, err := assetFiles.ReadFile("assets/" + request.Name)
	if err != nil {
		return assetResponse{statusCode: http.StatusInternalServerError, message: "asset unavailable"}, nil
	}
	return assetResponse{statusCode: http.StatusOK, contentType: contentType, body: content}, nil
}

func (*handler) liveReloadEvents(_ context.Context, _ *liveReloadRequest) (liveReloadResponse, error) {
	return liveReloadResponse{statusCode: http.StatusOK}, nil
}
