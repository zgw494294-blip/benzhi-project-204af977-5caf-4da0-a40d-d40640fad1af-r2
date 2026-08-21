package httpapi

import (
	"net/http"

	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/calibration"
)

func (s *Server) handleGetReviews(response http.ResponseWriter, sessionID string) {
	reviews, err := s.service.GetReviews(sessionID)
	if err != nil {
		writeServiceError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, map[string]interface{}{"reviews": reviews})
}

func (s *Server) handleSubmitReview(response http.ResponseWriter, request *http.Request, sessionID string) {
	if rejectNonJSON(response, request) {
		return
	}
	var input calibration.ReviewInput
	if err := decodeJSON(request, &input); err != nil {
		writeError(response, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	session, review, err := s.service.SubmitReviewContext(request.Context(), sessionID, input)
	if err != nil {
		writeServiceError(response, err)
		return
	}
	writeJSON(response, http.StatusCreated, map[string]interface{}{"session": session, "review": review})
}
