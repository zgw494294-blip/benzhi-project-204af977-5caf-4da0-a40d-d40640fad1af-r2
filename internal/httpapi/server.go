package httpapi

import (
	"net/http"
	"strings"

	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/calibration"
)

type Server struct {
	service *calibration.Service
}

func NewServer(service *calibration.Service) http.Handler {
	return &Server{service: service}
}

func (s *Server) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	path := strings.TrimSuffix(request.URL.Path, "/")
	if path == "" {
		path = "/"
	}
	if path == "/" {
		if request.Method != http.MethodGet {
			writeError(response, http.StatusMethodNotAllowed, "method_not_allowed", "页面只支持 GET")
			return
		}
		s.serveIndex(response)
		return
	}
	if strings.HasPrefix(path, "/static/") {
		s.serveStatic(response, request)
		return
	}
	if strings.HasPrefix(path, "/api/") {
		response.Header().Set("X-Auth-Mode", "placeholder")
		s.serveAPI(response, request, path)
		return
	}
	writeError(response, http.StatusNotFound, "not_found", "请求路径不存在")
}
