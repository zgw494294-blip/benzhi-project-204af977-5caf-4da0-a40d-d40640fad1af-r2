package httpapi

import (
	"net/http"
)

func (s *Server) serveIndex(response http.ResponseWriter) {
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := response.Write(indexHTML); err != nil {
		return
	}
}
