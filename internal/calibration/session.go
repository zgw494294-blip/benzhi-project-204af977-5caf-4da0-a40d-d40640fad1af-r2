package calibration

import (
	"context"
	"strings"
	"time"

	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/audit"
	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/domain"
	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/store"
)

func (s *Service) CreateSession(request CreateSessionRequest) (domain.CalibrationSession, []domain.ReferenceSample, error) {
	return s.CreateSessionContext(context.Background(), request)
}

func (s *Service) CreateSessionContext(ctx context.Context, request CreateSessionRequest) (domain.CalibrationSession, []domain.ReferenceSample, error) {
	request.DeviceID = strings.TrimSpace(request.DeviceID)
	request.DeviceName = strings.TrimSpace(request.DeviceName)
	request.ObservingBand = strings.TrimSpace(request.ObservingBand)
	request.Owner = strings.TrimSpace(request.Owner)
	if request.DeviceID == "" || request.DeviceName == "" || request.ObservingBand == "" || request.Owner == "" {
		return domain.CalibrationSession{}, nil, invalid("invalid_request", "设备编号、设备名称、观测波段和负责人均为必填")
	}
	if len(request.Samples) > 50 {
		return domain.CalibrationSession{}, nil, invalid("invalid_request", "单次最多登记 50 个标准样本")
	}
	now := s.now().UTC()
	sessionID := s.newID("SES")
	session := domain.CalibrationSession{ID: sessionID, DeviceID: request.DeviceID, DeviceName: request.DeviceName, ObservingBand: request.ObservingBand, Owner: request.Owner, Status: domain.StatusDraft, Version: 1, CreatedAt: now, UpdatedAt: now}
	samples := make([]domain.ReferenceSample, 0, len(request.Samples))
	seen := make(map[string]struct{})
	for _, input := range request.Samples {
		sample, err := s.makeSample(sessionID, input, now)
		if err != nil {
			return domain.CalibrationSession{}, nil, err
		}
		if _, exists := seen[sample.SampleNumber]; exists {
			return domain.CalibrationSession{}, nil, invalid("duplicate_sample", "样本编号不能重复")
		}
		seen[sample.SampleNumber] = struct{}{}
		samples = append(samples, sample)
	}
	if len(samples) > 0 {
		session.Status = domain.StatusMeasuring
	}
	if err := s.store.UpdateContext(ctx, func(ledger *store.Ledger) error {
		ledger.Sessions[session.ID] = session
		ledger.Samples[session.ID] = append([]domain.ReferenceSample(nil), samples...)
		audit.Append(&ledger.Audits, audit.NewEvent(s.newID("AUD"), session.ID, "session.created", session.Owner, map[string]string{"deviceID": session.DeviceID, "sampleCount": fmtInt(len(samples))}, now))
		if len(samples) > 0 {
			audit.Append(&ledger.Audits, audit.NewEvent(s.newID("AUD"), session.ID, "samples.registered", samples[0].RegisteredBy, map[string]string{"count": fmtInt(len(samples))}, now))
		}
		return nil
	}); err != nil {
		return domain.CalibrationSession{}, nil, err
	}
	return session, samples, nil
}

func (s *Service) RegisterSamples(sessionID string, request RegisterSamplesRequest) (domain.CalibrationSession, []domain.ReferenceSample, error) {
	return s.RegisterSamplesContext(context.Background(), sessionID, request)
}

func (s *Service) RegisterSamplesContext(ctx context.Context, sessionID string, request RegisterSamplesRequest) (domain.CalibrationSession, []domain.ReferenceSample, error) {
	if request.ExpectedVersion < 1 || len(request.Samples) == 0 {
		return domain.CalibrationSession{}, nil, invalid("invalid_request", "expectedVersion 和标准样本清单为必填")
	}
	now := s.now().UTC()
	created := make([]domain.ReferenceSample, 0, len(request.Samples))
	err := s.store.UpdateContext(ctx, func(ledger *store.Ledger) error {
		session, ok := ledger.Sessions[sessionID]
		if !ok {
			return notFound("校准会话")
		}
		if session.Version != request.ExpectedVersion {
			return invalid("version_conflict", fmtVersion(session.Version))
		}
		if !domain.CanRegisterSample(session.Status) {
			return invalid("invalid_state", "当前状态只能在草稿阶段登记样本")
		}
		seen := make(map[string]struct{})
		for _, existing := range ledger.Samples[sessionID] {
			seen[existing.SampleNumber] = struct{}{}
		}
		for _, input := range request.Samples {
			sample, err := s.makeSample(sessionID, input, now)
			if err != nil {
				return err
			}
			if _, exists := seen[sample.SampleNumber]; exists {
				return invalid("duplicate_sample", "样本编号不能与已有样本重复")
			}
			seen[sample.SampleNumber] = struct{}{}
			created = append(created, sample)
		}
		ledger.Samples[sessionID] = append(ledger.Samples[sessionID], created...)
		session.Status = domain.StatusMeasuring
		session.Version++
		session.UpdatedAt = now
		ledger.Sessions[sessionID] = session
		audit.Append(&ledger.Audits, audit.NewEvent(s.newID("AUD"), sessionID, "samples.registered", created[0].RegisteredBy, map[string]string{"count": fmtInt(len(created))}, now))
		return nil
	})
	if err != nil {
		return domain.CalibrationSession{}, nil, err
	}
	return s.GetSession(sessionID)
}

func (s *Service) makeSample(sessionID string, input SampleInput, now time.Time) (domain.ReferenceSample, error) {
	sample := domain.ReferenceSample{ID: s.newID("SMP"), SessionID: sessionID, SampleNumber: strings.TrimSpace(input.SampleNumber), ReferenceValue: input.ReferenceValue, Unit: strings.TrimSpace(input.Unit), AllowedDelta: input.AllowedDelta, RegisteredBy: strings.TrimSpace(input.RegisteredBy), RegisteredAt: now}
	if err := sample.Validate(); err != nil {
		return domain.ReferenceSample{}, err
	}
	return sample, nil
}
