package corruptledgererrorchain_test

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/domain"
	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/store"
)

func TestCorruptLedgerPreservesSentinelErrorChain(t *testing.T) {
	repository, err := store.New(filepath.Join(t.TempDir(), "ledger.json"))
	if err != nil {
		t.Fatal(err)
	}
	err = repository.Update(func(ledger *store.Ledger) error {
		ledger.Sessions["broken"] = domain.CalibrationSession{
			ID:            "broken",
			DeviceName:    "校准设备",
			ObservingBand: "可见光",
			Owner:         "工程师",
			Status:        domain.StatusDraft,
			Version:       1,
			CreatedAt:     testTime,
			UpdatedAt:     testTime,
		}
		return nil
	})
	if err == nil {
		t.Fatal("expected corrupt ledger to be rejected")
	}
	if !errors.Is(err, store.ErrCorruptLedger) {
		t.Fatalf("expected errors.Is(err, store.ErrCorruptLedger), got %v", err)
	}
}

var testTime = time.Date(2026, 8, 20, 8, 0, 0, 0, time.UTC)
