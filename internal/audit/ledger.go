package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"

	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/domain"
)

func Append(events *map[string][]domain.AuditEvent, event domain.AuditEvent) {
	chain := (*events)[event.SessionID]
	event.Sequence = len(chain) + 1
	if len(chain) > 0 {
		event.PreviousHash = chain[len(chain)-1].EventHash
	}
	event.EventHash = hashEvent(event)
	(*events)[event.SessionID] = append(chain, event)
}

func hashEvent(event domain.AuditEvent) string {
	payload := struct {
		ID           string            `json:"id"`
		SessionID    string            `json:"sessionID"`
		Action       string            `json:"action"`
		Actor        string            `json:"actor"`
		Details      map[string]string `json:"details"`
		Sequence     int               `json:"sequence"`
		OccurredAt   string            `json:"occurredAt"`
		PreviousHash string            `json:"previousHash"`
	}{event.ID, event.SessionID, event.Action, event.Actor, event.Details, event.Sequence, event.OccurredAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"), event.PreviousHash}
	encoded, _ := json.Marshal(payload)
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}

func ForSession(events map[string][]domain.AuditEvent, sessionID string) []domain.AuditEvent {
	result := events[sessionID]
	sort.SliceStable(result, func(i, j int) bool { return result[i].Sequence < result[j].Sequence })
	return result
}
