package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/calibration"
	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/domain"
)

type errorPayload struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeJSON(response http.ResponseWriter, status int, value interface{}) {
	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(value)
}

func writeError(response http.ResponseWriter, status int, code, message string) {
	writeJSON(response, status, errorPayload{Error: errorDetail{Code: code, Message: message}})
}

func decodeJSON(request *http.Request, target interface{}) error {
	request.Body = http.MaxBytesReader(nil, request.Body, 1<<20)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return errors.New("请求 JSON 格式无效")
	}
	var trailing interface{}
	if err := decoder.Decode(&trailing); err != io.EOF {
		return errors.New("请求 JSON 只能包含一个对象")
	}
	return nil
}

func writeServiceError(response http.ResponseWriter, err error) {
	var serviceError calibration.Error
	if errors.As(err, &serviceError) {
		status := http.StatusBadRequest
		switch serviceError.Code {
		case "not_found":
			status = http.StatusNotFound
		case "version_conflict", "idempotency_conflict", "sequence_conflict":
			status = http.StatusConflict
		}
		writeError(response, status, serviceError.Code, serviceError.Message)
		return
	}
	var ruleError domain.RuleError
	if errors.As(err, &ruleError) {
		writeError(response, http.StatusBadRequest, ruleError.Code, ruleError.Message)
		return
	}
	writeError(response, http.StatusInternalServerError, "internal_error", "服务内部错误")
}
