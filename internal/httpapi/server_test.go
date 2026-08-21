package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/calibration"
	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/store"
)

func TestServerProvidesHTMLAndUnifiedErrors(t *testing.T) {
	repository, err := store.New(filepath.Join(t.TempDir(), "ledger.json"))
	if err != nil {
		t.Fatal(err)
	}
	service := calibration.NewService(repository)
	server := httptest.NewServer(NewServer(service))
	defer server.Close()
	page, err := http.Get(server.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	page.Body.Close()
	if page.StatusCode != http.StatusOK || !strings.Contains(page.Header.Get("Content-Type"), "text/html") {
		t.Fatalf("unexpected page response: %d %s", page.StatusCode, page.Header.Get("Content-Type"))
	}
	response, err := http.Get(server.URL + "/api/sessions/missing")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusNotFound || body.Error.Code != "not_found" {
		t.Fatalf("unexpected error mapping: %d %#v", response.StatusCode, body)
	}
	if response.Header.Get("X-Auth-Mode") != "placeholder" {
		t.Fatal("expected authentication placeholder header")
	}
}

func TestServerFiltersSessionsAndRejectsInvalidStatus(t *testing.T) {
	repository, err := store.New(filepath.Join(t.TempDir(), "ledger.json"))
	if err != nil {
		t.Fatal(err)
	}
	service := calibration.NewService(repository)
	if _, _, err := service.CreateSession(calibration.CreateSessionRequest{DeviceID: "AST-HTTP-DRAFT", DeviceName: "草稿仪器", ObservingBand: "可见光", Owner: "工程师"}); err != nil {
		t.Fatal(err)
	}
	measuring, _, err := service.CreateSession(calibration.CreateSessionRequest{DeviceID: "AST-HTTP-MEASURE", DeviceName: "测量仪器", ObservingBand: "红外", Owner: "工程师", Samples: []calibration.SampleInput{{SampleNumber: "REF-1", ReferenceValue: 10, Unit: "ADU", AllowedDelta: 1, RegisteredBy: "登记员"}}})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(NewServer(service))
	defer server.Close()

	response, err := http.Get(server.URL + "/api/sessions?status=measuring&deviceID=" + measuring.DeviceID)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var filtered struct {
		Sessions []struct {
			ID          string          `json:"id"`
			Status      string          `json:"status"`
			Certificate json.RawMessage `json:"certificate"`
		} `json:"sessions"`
	}
	if err := json.NewDecoder(response.Body).Decode(&filtered); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || len(filtered.Sessions) != 1 || filtered.Sessions[0].ID != measuring.ID || filtered.Sessions[0].Status != "measuring" || filtered.Sessions[0].Certificate != nil {
		t.Fatalf("unexpected filtered response: %d %#v", response.StatusCode, filtered)
	}

	revision := repository.Snapshot().Revision
	response, err = http.Get(server.URL + "/api/sessions?status=invalid")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var invalidResponse errorPayload
	if err := json.NewDecoder(response.Body).Decode(&invalidResponse); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusBadRequest || invalidResponse.Error.Code != "invalid_status" || invalidResponse.Error.Message == "" {
		t.Fatalf("unexpected invalid status response: %d %#v", response.StatusCode, invalidResponse)
	}
	if got := repository.Snapshot().Revision; got != revision {
		t.Fatalf("invalid filter changed ledger revision from %d to %d", revision, got)
	}
}

func TestServerMapsDomainValidationAndRejectsTrailingJSON(t *testing.T) {
	repository, err := store.New(filepath.Join(t.TempDir(), "ledger.json"))
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(NewServer(calibration.NewService(repository)))
	defer server.Close()
	request, err := http.NewRequest(http.MethodPost, server.URL+"/api/sessions", bytes.NewBufferString(`{"deviceID":"AST-BAD","deviceName":"坏输入","observingBand":"可见光","owner":"工程师","samples":[{"sampleNumber":"REF-01","referenceValue":10,"unit":"ADU","allowedDelta":0,"registeredBy":"登记员"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var invalidResponse errorPayload
	if err := json.NewDecoder(response.Body).Decode(&invalidResponse); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusBadRequest || invalidResponse.Error.Code != "invalid_sample" {
		t.Fatalf("领域校验错误映射错误: %d %#v", response.StatusCode, invalidResponse)
	}
	request, err = http.NewRequest(http.MethodPost, server.URL+"/api/sessions", bytes.NewBufferString(`{"deviceID":"AST-TRAIL","deviceName":"尾随输入","observingBand":"可见光","owner":"工程师"} {}`))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err = http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("尾随 JSON 未被拒绝: %d", response.StatusCode)
	}
}
