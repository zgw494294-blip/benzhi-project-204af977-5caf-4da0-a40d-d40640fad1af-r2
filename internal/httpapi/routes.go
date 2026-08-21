package httpapi

import (
	"net/http"
	"strings"
)

func (s *Server) serveAPI(response http.ResponseWriter, request *http.Request, path string) {
	parts := splitPath(path)
	if len(parts) == 2 && parts[0] == "api" && parts[1] == "sessions" {
		if request.Method == http.MethodGet {
			if certificateNos, ok := request.URL.Query()["certificateNo"]; ok {
				certificateNo := ""
				if len(certificateNos) > 0 {
					certificateNo = certificateNos[0]
				}
				s.handleFindSessionsByCertificateNo(response, certificateNo)
				return
			}
			s.handleListSessions(response, request)
			return
		}
		if request.Method == http.MethodPost {
			s.handleCreateSession(response, request)
			return
		}
	}
	if len(parts) < 3 || parts[0] != "api" || parts[1] != "sessions" {
		writeError(response, http.StatusNotFound, "not_found", "API 路径不存在")
		return
	}
	sessionID := parts[2]
	if len(parts) == 3 && request.Method == http.MethodGet {
		s.handleGetSession(response, sessionID)
		return
	}
	if len(parts) != 4 {
		writeError(response, http.StatusNotFound, "not_found", "API 路径不存在")
		return
	}
	switch parts[3] {
	case "samples":
		if request.Method == http.MethodGet {
			s.handleGetSamples(response, sessionID)
			return
		}
		if request.Method == http.MethodPost {
			s.handleRegisterSamples(response, request, sessionID)
			return
		}
	case "measurements":
		if request.Method == http.MethodGet {
			s.handleGetMeasurements(response, sessionID)
			return
		}
		if request.Method == http.MethodPost {
			s.handleSubmitMeasurement(response, request, sessionID)
			return
		}
	case "review":
		if request.Method == http.MethodGet {
			s.handleGetReviews(response, sessionID)
			return
		}
		if request.Method == http.MethodPost {
			s.handleSubmitReview(response, request, sessionID)
			return
		}
	case "seal":
		if request.Method == http.MethodPost {
			s.handleSeal(response, request, sessionID)
			return
		}
	case "audit":
		if request.Method == http.MethodGet {
			s.handleGetAudit(response, sessionID)
			return
		}
	case "certificate":
		if request.Method == http.MethodGet {
			s.handleGetCertificate(response, sessionID)
			return
		}
	}
	writeError(response, http.StatusMethodNotAllowed, "method_not_allowed", "该 API 不支持当前请求方法")
}

func splitPath(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return nil
	}
	parts := strings.Split(trimmed, "/")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return nil
		}
	}
	return parts
}
