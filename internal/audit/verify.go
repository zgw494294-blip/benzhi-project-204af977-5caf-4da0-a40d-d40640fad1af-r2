package audit

import (
	"fmt"

	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/domain"
)

func Verify(events []domain.AuditEvent) error {
	var previous string
	for index, event := range events {
		if event.Sequence != index+1 {
			return fmt.Errorf("审计序列中断: 期望 %d，实际 %d", index+1, event.Sequence)
		}
		if event.PreviousHash != previous {
			return fmt.Errorf("审计前置哈希不匹配: %s", event.ID)
		}
		if hashEvent(event) != event.EventHash {
			return fmt.Errorf("审计事件哈希不匹配: %s", event.ID)
		}
		previous = event.EventHash
	}
	return nil
}
