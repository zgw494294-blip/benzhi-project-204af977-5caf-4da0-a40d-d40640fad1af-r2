package idempotentretryrevision

import (
	"path/filepath"
	"testing"

	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/calibration"
	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/store"
)

func TestIdempotentMeasurementRetryDoesNotAdvanceLedgerRevision(t *testing.T) {
	repository, err := store.New(filepath.Join(t.TempDir(), "ledger.json"))
	if err != nil {
		t.Fatal(err)
	}
	service := calibration.NewService(repository)
	session, samples, err := service.CreateSession(calibration.CreateSessionRequest{
		DeviceID:      "AST-IDEMPOTENT",
		DeviceName:    "幂等重试测试仪器",
		ObservingBand: "可见光",
		Owner:         "工程师",
		Samples: []calibration.SampleInput{{
			SampleNumber: "REF-1", ReferenceValue: 10, Unit: "ADU", AllowedDelta: 1, RegisteredBy: "登记员",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	request := calibration.MeasurementInput{
		ExpectedVersion: session.Version,
		SampleID:        samples[0].ID,
		MeasuredValue:   10,
		Operator:        "值班员",
		IdempotencyKey:  "retry-once",
	}
	session, original, err := service.SubmitMeasurement(session.ID, request)
	if err != nil {
		t.Fatal(err)
	}
	beforeRetry := repository.Snapshot()
	_, repeated, err := service.SubmitMeasurement(session.ID, request)
	if err != nil {
		t.Fatal(err)
	}
	afterRetry := repository.Snapshot()
	if repeated.ID != original.ID {
		t.Fatalf("幂等重试返回了不同记录: original=%s repeated=%s", original.ID, repeated.ID)
	}
	if afterRetry.Revision != beforeRetry.Revision {
		t.Fatalf("幂等重试不应推进账本 revision: before=%d after=%d", beforeRetry.Revision, afterRetry.Revision)
	}
	if afterRetry.Sessions[session.ID].Version != beforeRetry.Sessions[session.ID].Version {
		t.Fatalf("幂等重试不应改变会话版本: before=%d after=%d", beforeRetry.Sessions[session.ID].Version, afterRetry.Sessions[session.ID].Version)
	}
	if len(afterRetry.Audits[session.ID]) != len(beforeRetry.Audits[session.ID]) {
		t.Fatalf("幂等重试不应追加审计事件: before=%d after=%d", len(beforeRetry.Audits[session.ID]), len(afterRetry.Audits[session.ID]))
	}
}
