package httpapi

import (
	"net/http"
	"strings"
)

func requireJSON(request *http.Request) bool {
	return strings.HasPrefix(request.Header.Get("Content-Type"), "application/json")
}

func rejectNonJSON(response http.ResponseWriter, request *http.Request) bool {
	if !requireJSON(request) {
		writeError(response, http.StatusUnsupportedMediaType, "unsupported_media_type", "请求必须使用 application/json")
		return true
	}
	return false
}
