package audit

import (
	"testing"
	"time"

	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/domain"
)

func TestAppendBuildsVerifiableChain(t *testing.T) {
	ledger := map[string][]domain.AuditEvent{}
	now := time.Date(2026, 8, 20, 8, 0, 0, 0, time.UTC)
	Append(&ledger, NewEvent("a1", "s1", "session.created", "工程师", map[string]string{"deviceID": "AST-1"}, now))
	Append(&ledger, NewEvent("a2", "s1", "samples.registered", "工程师", map[string]string{"count": "2"}, now.Add(time.Second)))
	if err := Verify(ledger["s1"]); err != nil {
		t.Fatal(err)
	}
	ledger["s1"][1].Details["count"] = "9"
	if err := Verify(ledger["s1"]); err == nil {
		t.Fatal("expected tampered event to fail verification")
	}
}
