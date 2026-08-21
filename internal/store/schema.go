package store

import "benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/domain"

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
	output := Ledger{
		SchemaVersion: input.SchemaVersion,
		Revision:      input.Revision,
		Sessions:      make(map[string]domain.CalibrationSession, len(input.Sessions)),
		Samples:       make(map[string][]domain.ReferenceSample, len(input.Samples)),
		Measurements:  make(map[string][]domain.MeasurementRecord, len(input.Measurements)),
		Reviews:       make(map[string][]domain.QualityReview, len(input.Reviews)),
		Certificates:  make(map[string]domain.CalibrationCertificate, len(input.Certificates)),
		Audits:        make(map[string][]domain.AuditEvent, len(input.Audits)),
	}
	for id, session := range input.Sessions {
		output.Sessions[id] = session
	}
	for id, samples := range input.Samples {
		if samples == nil {
			output.Samples[id] = nil
			continue
		}
		copied := make([]domain.ReferenceSample, len(samples))
		copy(copied, samples)
		output.Samples[id] = copied
	}
	for id, measurements := range input.Measurements {
		if measurements == nil {
			output.Measurements[id] = nil
			continue
		}
		copied := make([]domain.MeasurementRecord, len(measurements))
		copy(copied, measurements)
		output.Measurements[id] = copied
	}
	for id, reviews := range input.Reviews {
		if reviews == nil {
			output.Reviews[id] = nil
			continue
		}
		copied := make([]domain.QualityReview, len(reviews))
		copy(copied, reviews)
		output.Reviews[id] = copied
	}
	for id, certificate := range input.Certificates {
		output.Certificates[id] = certificate
	}
	for id, events := range input.Audits {
		if events == nil {
			output.Audits[id] = nil
			continue
		}
		copied := make([]domain.AuditEvent, len(events))
		copy(copied, events)
		output.Audits[id] = copied
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
