package httpapi

import (
	"embed"
	"net/http"
	"path"
	"strings"
)

//go:embed web/index.html web/styles.css web/app.js
var frontendFiles embed.FS

var indexHTML = mustReadFrontend("web/index.html")

func mustReadFrontend(name string) []byte {
	contents, err := frontendFiles.ReadFile(name)
	if err != nil {
		panic(err)
	}
	return contents
}

func (s *Server) serveStatic(response http.ResponseWriter, request *http.Request) {
	name := strings.TrimPrefix(path.Clean(request.URL.Path), "/static/")
	var file string
	var contentType string
	switch name {
	case "styles.css":
		file, contentType = "web/styles.css", "text/css; charset=utf-8"
	case "app.js":
		file, contentType = "web/app.js", "text/javascript; charset=utf-8"
	default:
		writeError(response, http.StatusNotFound, "not_found", "静态资源不存在")
		return
	}
	contents, err := frontendFiles.ReadFile(file)
	if err != nil {
		writeError(response, http.StatusNotFound, "not_found", "静态资源不存在")
		return
	}
	response.Header().Set("Content-Type", contentType)
	_, _ = response.Write(contents)
}
