package store

import (
	"encoding/json"

	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/domain"
)

const CurrentSchemaVersion = 1

type Ledger struct {
	SchemaVersion int                                      `json:"schemaVersion"`
	Revision      int64                                    `json:"revision"`
	Sessions      map[string]domain.CalibrationSession     `json:"sessions"`
	Samples       map[string][]domain.ReferenceSample      `json:"samples"`
	Measurements  map[string][]domain.MeasurementRecord    `json:"measurements"`
	Reviews       map[string][]domain.QualityReview        `json:"reviews"`
	Certificates  map[string]domain.CalibrationCertificate `json:"certificates"`
	Audits        map[string][]domain.AuditEvent           `json:"audits"`
}

func NewLedger() Ledger {
	return Ledger{
		SchemaVersion: CurrentSchemaVersion,
		Sessions:      make(map[string]domain.CalibrationSession),
		Samples:       make(map[string][]domain.ReferenceSample),
		Measurements:  make(map[string][]domain.MeasurementRecord),
		Reviews:       make(map[string][]domain.QualityReview),
		Certificates:  make(map[string]domain.CalibrationCertificate),
		Audits:        make(map[string][]domain.AuditEvent),
	}
}

func cloneLedger(input Ledger) (Ledger, error) {
	encoded, err := json.Marshal(input)
	if err != nil {
		return Ledger{}, err
	}
	var output Ledger
	if err := json.Unmarshal(encoded, &output); err != nil {
		return Ledger{}, err
	}
	return output, nil
}

func normalizeLedger(input Ledger) Ledger {
	if input.Sessions == nil {
		input.Sessions = make(map[string]domain.CalibrationSession)
	}
	if input.Samples == nil {
		input.Samples = make(map[string][]domain.ReferenceSample)
	}
	if input.Measurements == nil {
		input.Measurements = make(map[string][]domain.MeasurementRecord)
	}
	if input.Reviews == nil {
		input.Reviews = make(map[string][]domain.QualityReview)
	}
	if input.Certificates == nil {
		input.Certificates = make(map[string]domain.CalibrationCertificate)
	}
	if input.Audits == nil {
		input.Audits = make(map[string][]domain.AuditEvent)
	}
	return input
}
