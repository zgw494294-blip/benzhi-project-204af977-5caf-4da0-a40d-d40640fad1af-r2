package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"

	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/calibration"
	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/store"
)

func RunSelfCheck() error {
	directory, err := os.MkdirTemp("", "star-calibration-selfcheck-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(directory)
	repository, err := store.New(filepath.Join(directory, "ledger.json"))
	if err != nil {
		return err
	}
	handler := NewServer(calibration.NewService(repository))
	server := httptest.NewServer(handler)
	defer server.Close()
	var created struct {
		Session struct {
			ID      string `json:"id"`
			Version int64  `json:"version"`
		} `json:"session"`
		Samples []struct {
			ID             string  `json:"id"`
			ReferenceValue float64 `json:"referenceValue"`
		} `json:"samples"`
	}
	if err := selfcheckRequest(server.Client(), server.URL+"/api/sessions", http.MethodPost, map[string]interface{}{
		"deviceID": "AST-SELF", "deviceName": "自检光谱仪", "observingBand": "450-700 nm", "owner": "自检工程师",
		"samples": []map[string]interface{}{{"sampleNumber": "REF-A", "referenceValue": 100.0, "unit": "ADU", "allowedDelta": 2.0, "registeredBy": "登记员"}, {"sampleNumber": "REF-B", "referenceValue": 50.0, "unit": "ADU", "allowedDelta": 1.0, "registeredBy": "登记员"}},
	}, &created, http.StatusCreated); err != nil {
		return fmt.Errorf("创建会话: %w", err)
	}
	if len(created.Samples) != 2 || created.Session.Version != 1 {
		return fmt.Errorf("创建结果不完整")
	}
	first := map[string]interface{}{"expectedVersion": created.Session.Version, "sampleID": created.Samples[0].ID, "measuredValue": 103.0, "operator": "值班员", "note": "自检超差", "idempotencyKey": "selfcheck-1"}
	var measured struct {
		Session struct {
			Version int64  `json:"version"`
			Status  string `json:"status"`
		} `json:"session"`
	}
	if err := selfcheckRequest(server.Client(), server.URL+"/api/sessions/"+created.Session.ID+"/measurements", http.MethodPost, first, &measured, http.StatusCreated); err != nil {
		return fmt.Errorf("提交首条测量: %w", err)
	}
	var repeated struct {
		Measurement struct {
			ID string `json:"id"`
		} `json:"measurement"`
	}
	if err := selfcheckRequest(server.Client(), server.URL+"/api/sessions/"+created.Session.ID+"/measurements", http.MethodPost, first, &repeated, http.StatusCreated); err != nil {
		return fmt.Errorf("幂等重试: %w", err)
	}
	if repeated.Measurement.ID == "" {
		return fmt.Errorf("幂等重试未返回原记录")
	}
	if err := selfcheckRequest(server.Client(), server.URL+"/api/sessions/"+created.Session.ID+"/measurements", http.MethodPost, map[string]interface{}{"expectedVersion": 1, "sampleID": created.Samples[1].ID, "measuredValue": 50.0, "operator": "值班员", "idempotencyKey": "stale"}, nil, http.StatusConflict); err != nil {
		return fmt.Errorf("版本冲突: %w", err)
	}
	second := map[string]interface{}{"expectedVersion": measured.Session.Version, "sampleID": created.Samples[1].ID, "measuredValue": 50.0, "operator": "值班员", "idempotencyKey": "selfcheck-2"}
	if err := selfcheckRequest(server.Client(), server.URL+"/api/sessions/"+created.Session.ID+"/measurements", http.MethodPost, second, &measured, http.StatusCreated); err != nil {
		return fmt.Errorf("提交末条初测: %w", err)
	}
	var reviewed struct {
		Session struct {
			Version int64  `json:"version"`
			Status  string `json:"status"`
		} `json:"session"`
	}
	if err := selfcheckRequest(server.Client(), server.URL+"/api/sessions/"+created.Session.ID+"/review", http.MethodPost, map[string]interface{}{"expectedVersion": measured.Session.Version, "reviewer": "质检员", "conclusion": "rework", "reworkReason": "首个样本超差"}, &reviewed, http.StatusCreated); err != nil {
		return fmt.Errorf("退回返工: %w", err)
	}
	if reviewed.Session.Status != "rework" {
		return fmt.Errorf("返工状态错误: %s", reviewed.Session.Status)
	}
	rework := map[string]interface{}{"expectedVersion": reviewed.Session.Version, "sampleID": created.Samples[0].ID, "measuredValue": 100.5, "operator": "值班员", "note": "返工补测", "idempotencyKey": "selfcheck-rework"}
	if err := selfcheckRequest(server.Client(), server.URL+"/api/sessions/"+created.Session.ID+"/measurements", http.MethodPost, rework, &measured, http.StatusCreated); err != nil {
		return fmt.Errorf("返工补测: %w", err)
	}
	if err := selfcheckRequest(server.Client(), server.URL+"/api/sessions/"+created.Session.ID+"/review", http.MethodPost, map[string]interface{}{"expectedVersion": measured.Session.Version, "reviewer": "质检员", "conclusion": "passed"}, &reviewed, http.StatusCreated); err != nil {
		return fmt.Errorf("复核通过: %w", err)
	}
	if reviewed.Session.Status != "ready_to_seal" {
		return fmt.Errorf("通过后状态错误: %s", reviewed.Session.Status)
	}
	var sealed struct {
		Session struct {
			Version int64  `json:"version"`
			Status  string `json:"status"`
		} `json:"session"`
		Certificate struct {
			CertificateNo string `json:"certificateNo"`
			SummaryHash   string `json:"summaryHash"`
		} `json:"certificate"`
	}
	if err := selfcheckRequest(server.Client(), server.URL+"/api/sessions/"+created.Session.ID+"/seal", http.MethodPost, map[string]interface{}{"expectedVersion": reviewed.Session.Version, "sealedBy": "负责人"}, &sealed, http.StatusCreated); err != nil {
		return fmt.Errorf("封存会话: %w", err)
	}
	if sealed.Session.Status != "sealed" || len(sealed.Certificate.SummaryHash) != 64 {
		return fmt.Errorf("封存结果不完整")
	}
	var bundle struct {
		AuditVerified bool `json:"auditVerified"`
		Certificate   struct {
			SummaryHash  string `json:"summaryHash"`
			Verification struct {
				Status          string `json:"status"`
				Verifiable      bool   `json:"verifiable"`
				SummaryVerified bool   `json:"summaryVerified"`
				AuditVerified   bool   `json:"auditVerified"`
			} `json:"verification"`
		} `json:"certificate"`
	}
	if err := selfcheckRequest(server.Client(), server.URL+"/api/sessions/"+created.Session.ID, http.MethodGet, nil, &bundle, http.StatusOK); err != nil {
		return fmt.Errorf("查询会话: %w", err)
	}
	if !bundle.AuditVerified || bundle.Certificate.SummaryHash == "" || bundle.Certificate.Verification.Status != "verified" || !bundle.Certificate.Verification.Verifiable || !bundle.Certificate.Verification.SummaryVerified || !bundle.Certificate.Verification.AuditVerified {
		return fmt.Errorf("查询结果审计或证书校验失败")
	}
	var certificateRead struct {
		Certificate struct {
			Verification struct {
				Verifiable bool `json:"verifiable"`
			} `json:"verification"`
		} `json:"certificate"`
	}
	if err := selfcheckRequest(server.Client(), server.URL+"/api/sessions/"+created.Session.ID+"/certificate", http.MethodGet, nil, &certificateRead, http.StatusOK); err != nil {
		return fmt.Errorf("查询证书: %w", err)
	}
	if !certificateRead.Certificate.Verification.Verifiable {
		return fmt.Errorf("证书查询未标记为可验证")
	}
	var lookup struct {
		Sessions []struct {
			ID            string `json:"id"`
			Status        string `json:"status"`
			AuditVerified bool   `json:"auditVerified"`
			Certificate   struct {
				SummaryHash  string `json:"summaryHash"`
				Verification struct {
					Status     string `json:"status"`
					Verifiable bool   `json:"verifiable"`
				} `json:"verification"`
			} `json:"certificate"`
		} `json:"sessions"`
	}
	lookupURL := server.URL + "/api/sessions?certificateNo=" + url.QueryEscape(sealed.Certificate.CertificateNo)
	if err := selfcheckRequest(server.Client(), lookupURL, http.MethodGet, nil, &lookup, http.StatusOK); err != nil {
		return fmt.Errorf("按证书编号查询: %w", err)
	}
	if len(lookup.Sessions) != 1 || lookup.Sessions[0].ID != created.Session.ID || lookup.Sessions[0].Status != "sealed" || lookup.Sessions[0].Certificate.SummaryHash != sealed.Certificate.SummaryHash || lookup.Sessions[0].Certificate.Verification.Status != "verified" || !lookup.Sessions[0].Certificate.Verification.Verifiable || !lookup.Sessions[0].AuditVerified {
		return fmt.Errorf("证书编号查询结果未通过校验")
	}
	var missing struct {
		Sessions []interface{} `json:"sessions"`
	}
	if err := selfcheckRequest(server.Client(), server.URL+"/api/sessions?certificateNo=missing", http.MethodGet, nil, &missing, http.StatusOK); err != nil {
		return fmt.Errorf("查询不存在的证书编号: %w", err)
	}
	if len(missing.Sessions) != 0 {
		return fmt.Errorf("不存在的证书编号返回了结果")
	}
	var filtered struct {
		Sessions []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"sessions"`
	}
	filterURL := server.URL + "/api/sessions?status=sealed&deviceID=" + url.QueryEscape("AST-SELF")
	if err := selfcheckRequest(server.Client(), filterURL, http.MethodGet, nil, &filtered, http.StatusOK); err != nil {
		return fmt.Errorf("按状态和设备编号查询: %w", err)
	}
	if len(filtered.Sessions) != 1 || filtered.Sessions[0].ID != created.Session.ID || filtered.Sessions[0].Status != "sealed" {
		return fmt.Errorf("状态和设备编号查询结果错误")
	}
	return nil
}

func selfcheckRequest(client *http.Client, url, method string, payload interface{}, target interface{}, expectedStatus int) error {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}
	request, err := http.NewRequest(method, url, body)
	if err != nil {
		return err
	}
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	contents, _ := io.ReadAll(response.Body)
	if response.StatusCode != expectedStatus {
		return fmt.Errorf("期望 HTTP %d，得到 %d: %s", expectedStatus, response.StatusCode, string(contents))
	}
	if target != nil && len(contents) > 0 {
		if err := json.Unmarshal(contents, target); err != nil {
			return err
		}
	}
	return nil
}
