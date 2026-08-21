package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/domain"
)

func TestUpdatePersistsAndReloadsLedger(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "ledger.json")
	repository, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	created := domain.CalibrationSession{ID: "s1", DeviceID: "AST-1", DeviceName: "光谱仪", ObservingBand: "可见光", Owner: "工程师", Status: domain.StatusDraft, Version: 1, CreatedAt: nowForTest(), UpdatedAt: nowForTest()}
	if err := repository.Update(func(ledger *Ledger) error { ledger.Sessions[created.ID] = created; return nil }); err != nil {
		t.Fatal(err)
	}
	reloaded, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := reloaded.Snapshot().Sessions["s1"].DeviceID; got != "AST-1" {
		t.Fatalf("expected persisted device ID, got %s", got)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("ledger file was not committed: %v", err)
	}
}

func TestUpdateRejectsInvalidCandidateWithoutChangingSnapshot(t *testing.T) {
	repository, err := New(filepath.Join(t.TempDir(), "ledger.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.Update(func(ledger *Ledger) error { ledger.Sessions["bad"] = domain.CalibrationSession{ID: "bad"}; return nil }); err == nil {
		t.Fatal("expected invalid candidate to be rejected")
	}
	if len(repository.Snapshot().Sessions) != 0 {
		t.Fatal("invalid candidate changed in-memory snapshot")
	}
}

func TestNewRecoversFromLastAtomicBackupWhenPrimaryIsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.json")
	repository, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	created := domain.CalibrationSession{ID: "s1", DeviceID: "AST-1", DeviceName: "光谱仪", ObservingBand: "可见光", Owner: "工程师", Status: domain.StatusDraft, Version: 1, CreatedAt: nowForTest(), UpdatedAt: nowForTest()}
	if err := repository.Update(func(ledger *Ledger) error { ledger.Sessions[created.ID] = created; return nil }); err != nil {
		t.Fatal(err)
	}
	if err := repository.Update(func(ledger *Ledger) error {
		session := ledger.Sessions[created.ID]
		session.Version = 2
		ledger.Sessions[created.ID] = session
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	recovered, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := recovered.Snapshot().Sessions[created.ID].Version; got != 1 {
		t.Fatalf("expected backup revision to be recovered, got %d", got)
	}
}

func TestSnapshotIsolatedFromStore(t *testing.T) {
	repository, err := New(filepath.Join(t.TempDir(), "ledger.json"))
	if err != nil {
		t.Fatal(err)
	}
	created := domain.CalibrationSession{ID: "s1", DeviceID: "AST-1", DeviceName: "光谱仪", ObservingBand: "可见光", Owner: "工程师", Status: domain.StatusMeasuring, Version: 1, CreatedAt: nowForTest(), UpdatedAt: nowForTest()}
	sample := domain.ReferenceSample{ID: "sample-1", SessionID: created.ID, SampleNumber: "REF-01", ReferenceValue: 10, Unit: "ADU", AllowedDelta: 1, RegisteredBy: "工程师", RegisteredAt: nowForTest()}
	if err := repository.Update(func(ledger *Ledger) error {
		ledger.Sessions[created.ID] = created
		ledger.Samples[created.ID] = []domain.ReferenceSample{sample}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	snapshot := repository.Snapshot()
	session := snapshot.Sessions[created.ID]
	session.DeviceName = "被外部修改"
	snapshot.Sessions[created.ID] = session
	snapshot.Samples[created.ID][0].SampleNumber = "TAMPERED"
	actual := repository.Snapshot()
	if actual.Sessions[created.ID].DeviceName != "光谱仪" || actual.Samples[created.ID][0].SampleNumber != "REF-01" {
		t.Fatal("外部修改了存储内部快照")
	}
}

func TestNewFallsBackWhenPrimarySchemaIsInvalid(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.json")
	repository, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	created := domain.CalibrationSession{ID: "s1", DeviceID: "AST-1", DeviceName: "光谱仪", ObservingBand: "可见光", Owner: "工程师", Status: domain.StatusDraft, Version: 1, CreatedAt: nowForTest(), UpdatedAt: nowForTest()}
	if err := repository.Update(func(ledger *Ledger) error { ledger.Sessions[created.ID] = created; return nil }); err != nil {
		t.Fatal(err)
	}
	if err := repository.Update(func(ledger *Ledger) error {
		session := ledger.Sessions[created.ID]
		session.Version++
		ledger.Sessions[created.ID] = session
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"schemaVersion":999}`), 0o600); err != nil {
		t.Fatal(err)
	}
	recovered, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := recovered.Snapshot().Sessions[created.ID]; !ok {
		t.Fatal("未从 schemaVersion 错误的主账本恢复有效副本")
	}
}

func TestUpdateContextStopsCanceledOperation(t *testing.T) {
	repository, err := New(filepath.Join(t.TempDir(), "ledger.json"))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	called := false
	if err := repository.UpdateContext(ctx, func(ledger *Ledger) error { called = true; return nil }); err == nil {
		t.Fatal("已取消上下文仍然提交了更新")
	}
	if called {
		t.Fatal("已取消上下文仍然调用了 mutator")
	}
}

func nowForTest() time.Time { return time.Date(2026, 8, 20, 8, 0, 0, 0, time.UTC) }
