package startup_error_chain

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/store"
)

func TestStartupPreservesPrimaryAndBackupValidationCauses(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.json")
	if err := os.WriteFile(path, []byte(`{"schemaVersion":999}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path+".bak", []byte(`{"schemaVersion":1,"sessions":{"broken":{"id":"broken"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := store.New(path)
	if err == nil {
		t.Fatal("expected startup to reject unusable primary and backup ledgers")
	}
	if !errors.Is(err, store.ErrSchemaVersion) {
		t.Fatalf("expected primary schema sentinel in startup error, got %v", err)
	}
	if !errors.Is(err, store.ErrCorruptLedger) {
		t.Fatalf("expected backup corruption sentinel in startup error, got %v", err)
	}
}
