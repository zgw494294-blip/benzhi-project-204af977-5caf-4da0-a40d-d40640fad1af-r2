package calibration

import (
	"context"
	"strings"

	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/audit"
	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/domain"
	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/store"
)

func (s *Service) SubmitMeasurement(sessionID string, request MeasurementInput) (domain.CalibrationSession, domain.MeasurementRecord, error) {
	return s.SubmitMeasurementContext(context.Background(), sessionID, request)
}

func (s *Service) SubmitMeasurementContext(ctx context.Context, sessionID string, request MeasurementInput) (domain.CalibrationSession, domain.MeasurementRecord, error) {
	request.SampleID = strings.TrimSpace(request.SampleID)
	request.Operator = strings.TrimSpace(request.Operator)
	request.IdempotencyKey = strings.TrimSpace(request.IdempotencyKey)
	if request.ExpectedVersion < 1 || request.SampleID == "" || request.Operator == "" || request.IdempotencyKey == "" {
		return domain.CalibrationSession{}, domain.MeasurementRecord{}, invalid("invalid_request", "expectedVersion、sampleID、operator 和 idempotencyKey 均为必填")
	}
	now := s.now().UTC()
	var sessionResult domain.CalibrationSession
	var record domain.MeasurementRecord
	err := s.store.UpdateContext(ctx, func(ledger *store.Ledger) error {
		session, ok := ledger.Sessions[sessionID]
		if !ok {
			return notFound("校准会话")
		}
		if session.Status == domain.StatusSealed {
			return invalid("invalid_state", "当前状态不允许录入测量")
		}
		for _, existing := range ledger.Measurements[sessionID] {
			if existing.IdempotencyKey == request.IdempotencyKey {
				if existing.SampleID == request.SampleID && existing.MeasuredValue == request.MeasuredValue && existing.Operator == request.Operator && existing.Note == request.Note {
					sessionResult = session
					record = existing
					return nil
				}
				return invalid("idempotency_conflict", "相同幂等键已经用于另一条测量记录")
			}
		}
		if session.Version != request.ExpectedVersion {
			return invalid("version_conflict", fmtVersion(session.Version))
		}
		if !domain.CanMeasure(session.Status) {
			return invalid("invalid_state", "当前状态不允许录入测量")
		}
		samples := ledger.Samples[sessionID]
		sample, ok := domain.FindSample(samples, request.SampleID)
		if !ok {
			return notFound("标准样本")
		}
		nextSampleID := domain.NextSampleID(samples, ledger.Measurements[sessionID])
		if nextSampleID == "" && session.Status == domain.StatusRework && len(samples) > 0 {
			nextSampleID = samples[0].ID
		}
		if nextSampleID != sample.ID {
			return invalid("sequence_conflict", "请按标准样本顺序录入，返工时先补测首个超差样本")
		}
		sequence := len(ledger.Measurements[sessionID]) + 1
		deviation := domain.DeviationFor(sample, request.MeasuredValue)
		record = domain.MeasurementRecord{ID: s.newID("MEA"), SessionID: sessionID, SampleID: sample.ID, MeasuredValue: request.MeasuredValue, MeasurementSequence: sequence, MeasuredAt: now, Operator: request.Operator, Note: strings.TrimSpace(request.Note), Deviation: deviation, WithinTolerance: abs(deviation) <= sample.AllowedDelta, IdempotencyKey: request.IdempotencyKey}
		if err := record.Validate(); err != nil {
			return err
		}
		ledger.Measurements[sessionID] = append(ledger.Measurements[sessionID], record)
		allSamplesReady := domain.AllSamplesMeasured(samples, ledger.Measurements[sessionID])
		if session.Status == domain.StatusRework {
			allSamplesReady = domain.AllSamplesQualified(samples, ledger.Measurements[sessionID])
		}
		if allSamplesReady {
			session.Status = domain.StatusPendingReview
		} else if session.Status != domain.StatusRework {
			session.Status = domain.StatusMeasuring
		}
		session.Version++
		session.UpdatedAt = now
		ledger.Sessions[sessionID] = session
		audit.Append(&ledger.Audits, audit.NewEvent(s.newID("AUD"), sessionID, "measurement.submitted", record.Operator, map[string]string{"sampleID": record.SampleID, "withinTolerance": fmtBool(record.WithinTolerance)}, now))
		sessionResult = session
		return nil
	})
	if err != nil {
		return domain.CalibrationSession{}, domain.MeasurementRecord{}, err
	}
	return sessionResult, record, nil
}

func abs(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}

func fmtBool(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
