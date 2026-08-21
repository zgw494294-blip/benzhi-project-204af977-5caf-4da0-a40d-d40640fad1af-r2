package httpapi

import "net/http"

func (s *Server) handleGetAudit(response http.ResponseWriter, sessionID string) {
	events, verified, err := s.service.GetAudit(sessionID)
	if err != nil {
		writeServiceError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, map[string]interface{}{"audit": events, "verified": verified})
}

func (s *Server) handleGetCertificate(response http.ResponseWriter, sessionID string) {
	certificate, err := s.service.GetCertificate(sessionID)
	if err != nil {
		writeServiceError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, map[string]interface{}{"certificate": certificate})
}
