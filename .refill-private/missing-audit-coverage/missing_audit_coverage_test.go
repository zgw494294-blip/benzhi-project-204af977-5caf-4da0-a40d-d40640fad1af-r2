package missingauditcoverage

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/audit"
	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/domain"
	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/store"
)

func TestStartupRejectsLedgerWithMissingSessionAuditChain(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.json")
	now := time.Date(2026, 8, 21, 8, 0, 0, 0, time.UTC)
	ledger := store.NewLedger()
	first := domain.CalibrationSession{
		ID:            "session-with-audit",
		DeviceID:      "AST-AUD-1",
		DeviceName:    "审计样本仪器",
		ObservingBand: "可见光",
		Owner:         "工程师",
		Status:        domain.StatusDraft,
		Version:       1,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	second := first
	second.ID = "session-without-audit"
	second.DeviceID = "AST-AUD-2"
	ledger.Sessions[first.ID] = first
	ledger.Sessions[second.ID] = second
	audit.Append(&ledger.Audits, audit.NewEvent("audit-1", first.ID, "session.created", first.Owner, map[string]string{"deviceID": first.DeviceID}, now))
	encoded, err := json.Marshal(ledger)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		t.Fatal(err)
	}

	_, err = store.New(path)
	if err == nil {
		t.Fatal("账本缺少一个会话的审计链时仍然成功启动")
	}
	if !errors.Is(err, store.ErrCorruptLedger) {
		t.Fatalf("期望识别为账本损坏，得到 %v", err)
	}
}
