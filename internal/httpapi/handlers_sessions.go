package httpapi

import (
	"net/http"

	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/calibration"
)

func (s *Server) handleListSessions(response http.ResponseWriter, request *http.Request) {
	query := request.URL.Query()
	statusValues, hasStatus := query["status"]
	deviceIDValues, hasDeviceID := query["deviceID"]
	filters := calibration.SessionListFilters{HasStatus: hasStatus, HasDeviceID: hasDeviceID}
	if hasStatus && len(statusValues) > 0 {
		filters.Status = statusValues[0]
	}
	if hasDeviceID && len(deviceIDValues) > 0 {
		filters.DeviceID = deviceIDValues[0]
	}
	sessions, err := s.service.ListSessionsByFilters(filters)
	if err != nil {
		writeServiceError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, map[string]interface{}{"sessions": sessions})
}

func (s *Server) handleFindSessionsByCertificateNo(response http.ResponseWriter, certificateNo string) {
	writeJSON(response, http.StatusOK, map[string]interface{}{"sessions": s.service.ListSessionsByCertificateNo(certificateNo)})
}

func (s *Server) handleCreateSession(response http.ResponseWriter, request *http.Request) {
	if rejectNonJSON(response, request) {
		return
	}
	var input calibration.CreateSessionRequest
	if err := decodeJSON(request, &input); err != nil {
		writeError(response, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	session, samples, err := s.service.CreateSessionContext(request.Context(), input)
	if err != nil {
		writeServiceError(response, err)
		return
	}
	writeJSON(response, http.StatusCreated, map[string]interface{}{"session": session, "samples": samples})
}

func (s *Server) handleGetSession(response http.ResponseWriter, sessionID string) {
	session, samples, measurements, reviews, certificate, events, verified, err := s.service.GetBundle(sessionID)
	if err != nil {
		writeServiceError(response, err)
		return
	}
	progress, err := s.service.GetProgress(sessionID)
	if err != nil {
		writeServiceError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, map[string]interface{}{"session": session, "samples": samples, "measurements": measurements, "reviews": reviews, "certificate": certificate, "audit": events, "auditVerified": verified, "progress": progress})
}
