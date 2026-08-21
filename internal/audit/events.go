package audit

import (
	"time"

	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/domain"
)

func NewEvent(id, sessionID, action, actor string, details map[string]string, occurredAt time.Time) domain.AuditEvent {
	return domain.AuditEvent{ID: id, SessionID: sessionID, Action: action, Actor: actor, Details: details, OccurredAt: occurredAt}
}
