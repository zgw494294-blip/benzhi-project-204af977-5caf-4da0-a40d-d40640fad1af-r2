package calibration

import (
	"fmt"
	"strings"

	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/audit"
	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/domain"
)

type SessionListFilters struct {
	Status      string
	HasStatus   bool
	DeviceID    string
	HasDeviceID bool
}

func (s *Service) GetSession(id string) (domain.CalibrationSession, []domain.ReferenceSample, error) {
	ledger := s.store.Snapshot()
	session, ok := ledger.Sessions[id]
	if !ok {
		return domain.CalibrationSession{}, nil, notFound("校准会话")
	}
	return session, append([]domain.ReferenceSample(nil), ledger.Samples[id]...), nil
}

func (s *Service) ListSessions() []domain.CalibrationSession {
	result, _ := s.ListSessionsByFilters(SessionListFilters{})
	return result
}

func (s *Service) ListSessionsByFilters(filters SessionListFilters) ([]domain.CalibrationSession, error) {
	var status domain.SessionStatus
	if filters.HasStatus {
		parsed, ok := domain.ParseSessionStatus(filters.Status)
		if !ok {
			return nil, invalid("invalid_status", "status 参数必须是 draft、measuring、pending_review、rework、ready_to_seal 或 sealed")
		}
		status = parsed
	}
	cacheKey := fmt.Sprintf("%t:%s:%t:%s", filters.HasStatus, filters.Status, filters.HasDeviceID, filters.DeviceID)
	ledger := s.store.Snapshot()
	s.cacheMu.Lock()
	if cached, ok := s.listCache[cacheKey]; ok {
		if cached.revision == ledger.Revision {
			result := cached.sessions
			s.cacheMu.Unlock()
			return result, nil
		}
	}
	s.cacheMu.Unlock()
	result := make([]domain.CalibrationSession, 0, len(ledger.Sessions))
	for _, session := range ledger.Sessions {
		if filters.HasStatus && session.Status != status {
			continue
		}
		if filters.HasDeviceID && session.DeviceID != filters.DeviceID {
			continue
		}
		result = append(result, session)
	}
	sortSessions(result)
	s.cacheMu.Lock()
	s.listCache[cacheKey] = sessionListCacheEntry{revision: ledger.Revision, sessions: result}
	s.cacheMu.Unlock()
	return result, nil
}

func (s *Service) ListSessionsByCertificateNo(certificateNo string) []domain.SessionCertificateLookup {
	ledger := s.store.Snapshot()
	result := make([]domain.SessionCertificateLookup, 0)
	for sessionID, session := range ledger.Sessions {
		if session.Status != domain.StatusSealed {
			continue
		}
		certificate, ok := ledger.Certificates[sessionID]
		if !ok || certificate.CertificateNo != certificateNo {
			continue
		}
		samples := ledger.Samples[sessionID]
		measurements := ledger.Measurements[sessionID]
		reviews := ledger.Reviews[sessionID]
		events := audit.ForSession(ledger.Audits, sessionID)
		view := certificateView(certificate, session, samples, measurements, reviews, events)
		result = append(result, domain.SessionCertificateLookup{
			CalibrationSession: session,
			Certificate:        view,
			AuditVerified:      view.Verification.AuditVerified,
		})
	}
	sortSessionCertificateLookups(result)
	return result
}

func (s *Service) GetMeasurements(id string) ([]domain.MeasurementRecord, error) {
	ledger := s.store.Snapshot()
	if _, ok := ledger.Sessions[id]; !ok {
		return nil, notFound("校准会话")
	}
	return append([]domain.MeasurementRecord(nil), ledger.Measurements[id]...), nil
}

func (s *Service) GetReviews(id string) ([]domain.QualityReview, error) {
	ledger := s.store.Snapshot()
	if _, ok := ledger.Sessions[id]; !ok {
		return nil, notFound("校准会话")
	}
	return append([]domain.QualityReview(nil), ledger.Reviews[id]...), nil
}

func (s *Service) GetCertificate(id string) (domain.CertificateView, error) {
	ledger := s.store.Snapshot()
	session, ok := ledger.Sessions[id]
	if !ok {
		return domain.CertificateView{}, notFound("校准会话")
	}
	certificate, ok := ledger.Certificates[id]
	if !ok {
		return domain.CertificateView{}, notFound("校准证书")
	}
	samples := ledger.Samples[id]
	measurements := ledger.Measurements[id]
	reviews := ledger.Reviews[id]
	events := audit.ForSession(ledger.Audits, id)
	return certificateView(certificate, session, samples, measurements, reviews, events), nil
}

func (s *Service) GetAudit(id string) ([]domain.AuditEvent, bool, error) {
	ledger := s.store.Snapshot()
	if _, ok := ledger.Sessions[id]; !ok {
		return nil, false, notFound("校准会话")
	}
	events := audit.ForSession(ledger.Audits, id)
	return events, verifyAuditChain(events) == nil, nil
}

func (s *Service) GetBundle(id string) (domain.CalibrationSession, []domain.ReferenceSample, []domain.MeasurementRecord, []domain.QualityReview, *domain.CertificateView, []domain.AuditEvent, bool, error) {
	ledger := s.store.Snapshot()
	session, ok := ledger.Sessions[id]
	if !ok {
		return domain.CalibrationSession{}, nil, nil, nil, nil, nil, false, notFound("校准会话")
	}
	var certificate *domain.CertificateView
	samples := append([]domain.ReferenceSample(nil), ledger.Samples[id]...)
	measurements := append([]domain.MeasurementRecord(nil), ledger.Measurements[id]...)
	reviews := append([]domain.QualityReview(nil), ledger.Reviews[id]...)
	events := audit.ForSession(ledger.Audits, id)
	if value, exists := ledger.Certificates[id]; exists {
		view := certificateView(value, session, samples, measurements, reviews, events)
		certificate = &view
	}
	return session, samples, measurements, reviews, certificate, events, verifyAuditChain(events) == nil, nil
}

func certificateView(certificate domain.CalibrationCertificate, session domain.CalibrationSession, samples []domain.ReferenceSample, measurements []domain.MeasurementRecord, reviews []domain.QualityReview, events []domain.AuditEvent) domain.CertificateView {
	verification := domain.CertificateVerification{Status: domain.CertificateVerificationVerified, Verifiable: true, SummaryVerified: true, AuditVerified: true}
	var reasons []string
	if err := domain.VerifyCertificateDigest(certificate, session, samples, measurements, reviews); err != nil {
		verification.SummaryVerified = false
		reasons = append(reasons, err.Error())
	}
	if err := verifyAuditChain(events); err != nil {
		verification.AuditVerified = false
		reasons = append(reasons, fmt.Sprintf("审计哈希链校验失败: %v", err))
	}
	if !verification.SummaryVerified || !verification.AuditVerified {
		verification.Status = domain.CertificateVerificationInvalid
		verification.Verifiable = false
		verification.FailureReason = strings.Join(reasons, "；")
	}
	return domain.CertificateView{CalibrationCertificate: certificate, Verification: verification}
}

func verifyAuditChain(events []domain.AuditEvent) error {
	if len(events) == 0 {
		return fmt.Errorf("审计哈希链为空")
	}
	return audit.Verify(events)
}
