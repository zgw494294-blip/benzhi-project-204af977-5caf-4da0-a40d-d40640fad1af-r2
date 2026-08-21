package httpapi

import (
	"net/http"

	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/calibration"
)

func (s *Server) handleGetSamples(response http.ResponseWriter, sessionID string) {
	_, samples, err := s.service.GetSession(sessionID)
	if err != nil {
		writeServiceError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, map[string]interface{}{"samples": samples})
}

func (s *Server) handleRegisterSamples(response http.ResponseWriter, request *http.Request, sessionID string) {
	if rejectNonJSON(response, request) {
		return
	}
	var input calibration.RegisterSamplesRequest
	if err := decodeJSON(request, &input); err != nil {
		writeError(response, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	session, samples, err := s.service.RegisterSamplesContext(request.Context(), sessionID, input)
	if err != nil {
		writeServiceError(response, err)
		return
	}
	writeJSON(response, http.StatusCreated, map[string]interface{}{"session": session, "samples": samples})
}
