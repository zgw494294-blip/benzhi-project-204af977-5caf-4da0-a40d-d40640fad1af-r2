package httpapi

import (
	"net/http"

	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/calibration"
)

func (s *Server) handleSeal(response http.ResponseWriter, request *http.Request, sessionID string) {
	if rejectNonJSON(response, request) {
		return
	}
	var input calibration.SealInput
	if err := decodeJSON(request, &input); err != nil {
		writeError(response, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	session, certificate, err := s.service.SealSessionContext(request.Context(), sessionID, input)
	if err != nil {
		writeServiceError(response, err)
		return
	}
	writeJSON(response, http.StatusCreated, map[string]interface{}{"session": session, "certificate": certificate})
}
