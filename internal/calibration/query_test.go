package calibration

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/domain"
	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/store"
)

func TestListSessionsByFiltersMatchesStatusAndDeviceWithoutWriting(t *testing.T) {
	repository, err := store.New(filepath.Join(t.TempDir(), "ledger.json"))
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(repository)
	draft, _, err := service.CreateSession(CreateSessionRequest{DeviceID: "AST-DRAFT", DeviceName: "草稿仪器", ObservingBand: "可见光", Owner: "工程师"})
	if err != nil {
		t.Fatal(err)
	}
	measuring, _, err := service.CreateSession(CreateSessionRequest{DeviceID: "AST-MEASURE", DeviceName: "测量仪器", ObservingBand: "红外", Owner: "工程师", Samples: []SampleInput{{SampleNumber: "REF-1", ReferenceValue: 10, Unit: "ADU", AllowedDelta: 1, RegisteredBy: "登记员"}}})
	if err != nil {
		t.Fatal(err)
	}
	otherMeasuring, _, err := service.CreateSession(CreateSessionRequest{DeviceID: "AST-OTHER", DeviceName: "另一测量仪器", ObservingBand: "射电", Owner: "工程师", Samples: []SampleInput{{SampleNumber: "REF-1", ReferenceValue: 20, Unit: "ADU", AllowedDelta: 1, RegisteredBy: "登记员"}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.Update(func(ledger *store.Ledger) error {
		first := ledger.Sessions[draft.ID]
		first.UpdatedAt = time.Date(2026, 8, 20, 8, 0, 0, 0, time.UTC)
		ledger.Sessions[draft.ID] = first
		second := ledger.Sessions[measuring.ID]
		second.UpdatedAt = time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
		ledger.Sessions[measuring.ID] = second
		third := ledger.Sessions[otherMeasuring.ID]
		third.UpdatedAt = time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC)
		ledger.Sessions[otherMeasuring.ID] = third
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	all, err := service.ListSessionsByFilters(SessionListFilters{})
	if err != nil || len(all) != 3 || all[0].ID != measuring.ID || all[1].ID != otherMeasuring.ID || all[2].ID != draft.ID {
		t.Fatalf("expected all sessions in updatedAt descending order, err=%v sessions=%#v", err, all)
	}
	filtered, err := service.ListSessionsByFilters(SessionListFilters{Status: string(domain.StatusMeasuring), HasStatus: true, DeviceID: measuring.DeviceID, HasDeviceID: true})
	if err != nil || len(filtered) != 1 || filtered[0].ID != measuring.ID {
		t.Fatalf("expected status and device AND filtering, err=%v sessions=%#v", err, filtered)
	}

	revision := repository.Snapshot().Revision
	_, err = service.ListSessionsByFilters(SessionListFilters{Status: "unknown", HasStatus: true})
	var serviceError Error
	if !errors.As(err, &serviceError) || serviceError.Code != "invalid_status" {
		t.Fatalf("expected invalid status error, got %v", err)
	}
	if got := repository.Snapshot().Revision; got != revision {
		t.Fatalf("invalid lookup changed ledger revision from %d to %d", revision, got)
	}
}

func TestListSessionsByFiltersReflectsSessionsCreatedAfterInitialQuery(t *testing.T) {
	repository, err := store.New(filepath.Join(t.TempDir(), "ledger.json"))
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(repository)

	initial, err := service.ListSessionsByFilters(SessionListFilters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(initial) != 0 {
		t.Fatalf("expected empty initial session list, got %#v", initial)
	}

	created, _, err := service.CreateSession(CreateSessionRequest{
		DeviceID:      "AST-CACHE",
		DeviceName:    "缓存回归仪器",
		ObservingBand: "可见光",
		Owner:         "工程师",
	})
	if err != nil {
		t.Fatal(err)
	}

	updated, err := service.ListSessionsByFilters(SessionListFilters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(updated) != 1 || updated[0].ID != created.ID {
		t.Fatalf("expected newly created session after cached initial query, got %#v", updated)
	}
}

func TestGetCertificateReportsSnapshotIntegrity(t *testing.T) {
	repository, err := store.New(filepath.Join(t.TempDir(), "ledger.json"))
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(repository)
	session, samples, err := service.CreateSession(CreateSessionRequest{
		DeviceID:      "AST-VERIFY",
		DeviceName:    "证书校验光谱仪",
		ObservingBand: "可见光",
		Owner:         "负责人",
		Samples:       []SampleInput{{SampleNumber: "REF-1", ReferenceValue: 100, Unit: "ADU", AllowedDelta: 1, RegisteredBy: "登记员"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, _, err = service.SubmitMeasurement(session.ID, MeasurementInput{ExpectedVersion: session.Version, SampleID: samples[0].ID, MeasuredValue: 100, Operator: "值班员", IdempotencyKey: "verify-1"})
	if err != nil {
		t.Fatal(err)
	}
	session, _, err = service.SubmitReview(session.ID, ReviewInput{ExpectedVersion: session.Version, Reviewer: "质检员", Conclusion: "passed"})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = service.SealSession(session.ID, SealInput{ExpectedVersion: session.Version, SealedBy: "负责人"}); err != nil {
		t.Fatal(err)
	}

	view, err := service.GetCertificate(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !view.Verification.Verifiable || view.Verification.Status != "verified" {
		t.Fatalf("expected verified certificate, got %#v", view.Verification)
	}
	certificateNo := view.CertificateNo
	revisionBeforeMissingLookup := repository.Snapshot().Revision
	if results := service.ListSessionsByCertificateNo("missing"); len(results) != 0 {
		t.Fatalf("expected no certificate lookup results, got %#v", results)
	}
	if got := repository.Snapshot().Revision; got != revisionBeforeMissingLookup {
		t.Fatalf("missing lookup changed ledger revision from %d to %d", revisionBeforeMissingLookup, got)
	}
	results := service.ListSessionsByCertificateNo(certificateNo)
	if len(results) != 1 || results[0].ID != session.ID || results[0].Certificate.SummaryHash != view.SummaryHash || results[0].Certificate.Verification.Status != "verified" || !results[0].AuditVerified {
		t.Fatalf("expected verified certificate lookup, got %#v", results)
	}

	if err := repository.Update(func(ledger *store.Ledger) error {
		ledger.Measurements[session.ID][0].MeasuredValue = 100.5
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	view, err = service.GetCertificate(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if view.Verification.Verifiable || view.Verification.SummaryVerified || !view.Verification.AuditVerified || !strings.Contains(view.Verification.FailureReason, "摘要") {
		t.Fatalf("expected summary mismatch, got %#v", view.Verification)
	}
	results = service.ListSessionsByCertificateNo(certificateNo)
	if len(results) != 1 || results[0].Certificate.Verification.Status != "invalid" || results[0].Certificate.Verification.SummaryVerified || !results[0].Certificate.Verification.AuditVerified || !results[0].AuditVerified {
		t.Fatalf("expected invalid summary lookup, got %#v", results)
	}

	if err := repository.Update(func(ledger *store.Ledger) error {
		ledger.Measurements[session.ID][0].MeasuredValue = 100
		ledger.Audits[session.ID][0].Details["deviceID"] = "TAMPERED"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	view, err = service.GetCertificate(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if view.Verification.Verifiable || !view.Verification.SummaryVerified || view.Verification.AuditVerified || !strings.Contains(view.Verification.FailureReason, "审计") {
		t.Fatalf("expected audit mismatch, got %#v", view.Verification)
	}
	results = service.ListSessionsByCertificateNo(certificateNo)
	if len(results) != 1 || results[0].Certificate.Verification.Status != "invalid" || !results[0].Certificate.Verification.SummaryVerified || results[0].Certificate.Verification.AuditVerified || results[0].AuditVerified {
		t.Fatalf("expected invalid audit lookup, got %#v", results)
	}
}
