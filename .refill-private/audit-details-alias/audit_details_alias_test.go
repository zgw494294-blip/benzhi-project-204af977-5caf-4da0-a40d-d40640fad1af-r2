package auditdetailsalias_test

import (
	"path/filepath"
	"testing"

	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/calibration"
	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/store"
)

func TestAuditQueryDoesNotExposeMutableDetails(t *testing.T) {
	repository, err := store.New(filepath.Join(t.TempDir(), "ledger.json"))
	if err != nil {
		t.Fatal(err)
	}
	service := calibration.NewService(repository)
	session, _, err := service.CreateSession(calibration.CreateSessionRequest{
		DeviceID:      "AST-AUDIT-ALIAS",
		DeviceName:    "审计详情隔离仪器",
		ObservingBand: "可见光",
		Owner:         "工程师",
	})
	if err != nil {
		t.Fatal(err)
	}

	events, verified, err := service.GetAudit(session.ID)
	if err != nil || !verified || len(events) == 0 {
		t.Fatalf("expected a verified audit chain, events=%#v verified=%v err=%v", events, verified, err)
	}
	events[0].Details["deviceID"] = "外部篡改"

	refetched, verified, err := service.GetAudit(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !verified || refetched[0].Details["deviceID"] != "AST-AUDIT-ALIAS" {
		t.Fatalf("audit details mutation leaked into the store: events=%#v verified=%v", refetched, verified)
	}
}
