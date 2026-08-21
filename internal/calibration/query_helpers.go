package calibration

import (
	"fmt"
	"sort"

	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/domain"
)

func sortSessions(sessions []domain.CalibrationSession) {
	sort.SliceStable(sessions, func(i, j int) bool { return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt) })
}

func sortSessionCertificateLookups(lookups []domain.SessionCertificateLookup) {
	sort.SliceStable(lookups, func(i, j int) bool {
		return lookups[i].UpdatedAt.After(lookups[j].UpdatedAt)
	})
}

func formatFloat(value float64) string {
	return fmt.Sprintf("%.4f", value)
}
