package primary_ledger_data_loss

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/store"
)

func TestCorruptPrimaryWithoutBackupIsNotSilentlyReplaced(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.json")
	if err := os.WriteFile(path, []byte(`{"schemaVersion":999}`), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := store.New(path)
	if err == nil {
		t.Fatal("损坏的主账本在没有恢复副本时被静默替换为空账本")
	}
	if !errors.Is(err, store.ErrSchemaVersion) {
		t.Fatalf("启动错误未保留 schemaVersion 哨兵: %v", err)
	}
}
