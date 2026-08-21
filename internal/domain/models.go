package domain

import "time"

type CalibrationSession struct {
	ID            string        `json:"id"`
	DeviceID      string        `json:"deviceID"`
	DeviceName    string        `json:"deviceName"`
	ObservingBand string        `json:"observingBand"`
	Owner         string        `json:"owner"`
	Status        SessionStatus `json:"status"`
	Version       int64         `json:"version"`
	CreatedAt     time.Time     `json:"createdAt"`
	UpdatedAt     time.Time     `json:"updatedAt"`
}

type ReferenceSample struct {
	ID             string    `json:"id"`
	SessionID      string    `json:"sessionID"`
	SampleNumber   string    `json:"sampleNumber"`
	ReferenceValue float64   `json:"referenceValue"`
	Unit           string    `json:"unit"`
	AllowedDelta   float64   `json:"allowedDelta"`
	RegisteredBy   string    `json:"registeredBy"`
	RegisteredAt   time.Time `json:"registeredAt"`
}

type MeasurementRecord struct {
	ID                  string    `json:"id"`
	SessionID           string    `json:"sessionID"`
	SampleID            string    `json:"sampleID"`
	MeasuredValue       float64   `json:"measuredValue"`
	MeasurementSequence int       `json:"measurementSequence"`
	MeasuredAt          time.Time `json:"measuredAt"`
	Operator            string    `json:"operator"`
	Note                string    `json:"note"`
	Deviation           float64   `json:"deviation"`
	WithinTolerance     bool      `json:"withinTolerance"`
	IdempotencyKey      string    `json:"idempotencyKey"`
}

type QualityReview struct {
	ID               string    `json:"id"`
	SessionID        string    `json:"sessionID"`
	Reviewer         string    `json:"reviewer"`
	Conclusion       string    `json:"conclusion"`
	DeviationSummary string    `json:"deviationSummary"`
	ReworkReason     string    `json:"reworkReason"`
	ReviewedAt       time.Time `json:"reviewedAt"`
}

type CalibrationCertificate struct {
	ID            string    `json:"id"`
	SessionID     string    `json:"sessionID"`
	CertificateNo string    `json:"certificateNo"`
	SummaryHash   string    `json:"summaryHash"`
	SealedAt      time.Time `json:"sealedAt"`
	SealedBy      string    `json:"sealedBy"`
}

type AuditEvent struct {
	ID           string            `json:"id"`
	SessionID    string            `json:"sessionID"`
	Action       string            `json:"action"`
	Actor        string            `json:"actor"`
	Details      map[string]string `json:"details"`
	Sequence     int               `json:"sequence"`
	OccurredAt   time.Time         `json:"occurredAt"`
	PreviousHash string            `json:"previousHash"`
	EventHash    string            `json:"eventHash"`
}
