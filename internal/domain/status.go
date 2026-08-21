package domain

type SessionStatus string

const (
	StatusDraft         SessionStatus = "draft"
	StatusMeasuring     SessionStatus = "measuring"
	StatusPendingReview SessionStatus = "pending_review"
	StatusRework        SessionStatus = "rework"
	StatusReadyToSeal   SessionStatus = "ready_to_seal"
	StatusSealed        SessionStatus = "sealed"
)

func ParseSessionStatus(value string) (SessionStatus, bool) {
	status := SessionStatus(value)
	switch status {
	case StatusDraft, StatusMeasuring, StatusPendingReview, StatusRework, StatusReadyToSeal, StatusSealed:
		return status, true
	default:
		return "", false
	}
}

func (s SessionStatus) Label() string {
	switch s {
	case StatusDraft:
		return "草稿"
	case StatusMeasuring:
		return "测量中"
	case StatusPendingReview:
		return "待复核"
	case StatusRework:
		return "返工"
	case StatusReadyToSeal:
		return "可封存"
	case StatusSealed:
		return "已封存"
	default:
		return "未知"
	}
}

func (s SessionStatus) Writable() bool {
	return s != StatusSealed
}

func CanRegisterSample(status SessionStatus) bool {
	return status == StatusDraft
}

func CanMeasure(status SessionStatus) bool {
	return status == StatusMeasuring || status == StatusRework
}

func CanReview(status SessionStatus) bool {
	return status == StatusPendingReview
}

func CanSeal(status SessionStatus) bool {
	return status == StatusReadyToSeal
}
