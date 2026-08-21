package calibration

import (
	"fmt"
	"strings"

	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/domain"
)

func fmtInt(value int) string {
	return fmt.Sprintf("%d", value)
}

func fmtVersion(value int64) string {
	return fmt.Sprintf("版本冲突：当前版本为 %d，请刷新后重试", value)
}

func normalizeConclusion(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "passed", "pass", "通过":
		return "passed"
	case "rework", "退回", "返工":
		return "rework"
	default:
		return ""
	}
}

func findReview(reviews []domain.QualityReview, conclusion string) bool {
	for _, review := range reviews {
		if review.Conclusion == conclusion {
			return true
		}
	}
	return false
}
