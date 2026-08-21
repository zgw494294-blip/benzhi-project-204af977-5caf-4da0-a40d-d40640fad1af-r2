package httpapi

import (
	"net/http"

	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/calibration"
)

func (s *Server) handleGetMeasurements(response http.ResponseWriter, sessionID string) {
	measurements, err := s.service.GetMeasurements(sessionID)
	if err != nil {
		writeServiceError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, map[string]interface{}{"measurements": measurements})
}

func (s *Server) handleSubmitMeasurement(response http.ResponseWriter, request *http.Request, sessionID string) {
	if rejectNonJSON(response, request) {
		return
	}
	var input calibration.MeasurementInput
	if err := decodeJSON(request, &input); err != nil {
		writeError(response, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	session, measurement, err := s.service.SubmitMeasurementContext(request.Context(), sessionID, input)
	if err != nil {
		writeServiceError(response, err)
		return
	}
	writeJSON(response, http.StatusCreated, map[string]interface{}{"session": session, "measurement": measurement})
}
