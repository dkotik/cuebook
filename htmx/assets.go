package htmx

import "net/http"

func serveAsset(w http.ResponseWriter, r *http.Request) {
	var contentType string
	switch name := r.PathValue("name"); name {
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
		http.NotFound(w, r)
		return
	}

	content, err := assetFiles.ReadFile("assets/" + r.PathValue("name"))
	if err != nil {
		http.Error(w, "asset unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = w.Write(content)
}
