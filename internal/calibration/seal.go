package calibration

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/audit"
	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/domain"
	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/store"
)

func (s *Service) SealSession(sessionID string, request SealInput) (domain.CalibrationSession, domain.CalibrationCertificate, error) {
	return s.SealSessionContext(context.Background(), sessionID, request)
}

func (s *Service) SealSessionContext(ctx context.Context, sessionID string, request SealInput) (domain.CalibrationSession, domain.CalibrationCertificate, error) {
	request.SealedBy = strings.TrimSpace(request.SealedBy)
	if request.ExpectedVersion < 1 || request.SealedBy == "" {
		return domain.CalibrationSession{}, domain.CalibrationCertificate{}, invalid("invalid_request", "expectedVersion 和 sealedBy 均为必填")
	}
	now := s.now().UTC()
	var sessionResult domain.CalibrationSession
	var certificate domain.CalibrationCertificate
	err := s.store.UpdateContext(ctx, func(ledger *store.Ledger) error {
		session, ok := ledger.Sessions[sessionID]
		if !ok {
			return notFound("校准会话")
		}
		if session.Status == domain.StatusSealed {
			certificate, ok = ledger.Certificates[sessionID]
			if !ok {
				return invalid("invalid_ledger", "已封存会话缺少证书")
			}
			sessionResult = session
			return nil
		}
		if session.Version != request.ExpectedVersion {
			return invalid("version_conflict", fmtVersion(session.Version))
		}
		if !domain.CanSeal(session.Status) {
			return invalid("invalid_state", "只有复核通过的会话才能封存")
		}
		samples := ledger.Samples[sessionID]
		measurements := ledger.Measurements[sessionID]
		if !domain.AllSamplesQualified(samples, measurements) || !findReview(ledger.Reviews[sessionID], "passed") {
			return invalid("quality_failed", "会话未满足全部样本合格和复核通过条件")
		}
		session.Status = domain.StatusSealed
		session.Version++
		session.UpdatedAt = now
		digest, err := domain.CertificateDigest(session, samples, measurements, ledger.Reviews[sessionID])
		if err != nil {
			return invalid("certificate_failed", "生成证书摘要失败")
		}
		certificate = domain.CalibrationCertificate{ID: s.newID("CRT"), SessionID: sessionID, CertificateNo: nextCertificateNumber(ledger, now), SummaryHash: digest, SealedAt: now, SealedBy: request.SealedBy}
		if err := certificate.Validate(); err != nil {
			return err
		}
		ledger.Certificates[sessionID] = certificate
		ledger.Sessions[sessionID] = session
		audit.Append(&ledger.Audits, audit.NewEvent(s.newID("AUD"), sessionID, "session.sealed", request.SealedBy, map[string]string{"certificateNo": certificate.CertificateNo, "summaryHash": digest}, now))
		sessionResult = session
		return nil
	})
	if err != nil {
		return domain.CalibrationSession{}, domain.CalibrationCertificate{}, err
	}
	return sessionResult, certificate, nil
}

func nextCertificateNumber(ledger *store.Ledger, now time.Time) string {
	prefix := "CAL-" + now.Format("20060102") + "-"
	highest := 0
	for _, certificate := range ledger.Certificates {
		if !strings.HasPrefix(certificate.CertificateNo, prefix) {
			continue
		}
		number, err := strconv.Atoi(strings.TrimPrefix(certificate.CertificateNo, prefix))
		if err == nil && number > highest {
			highest = number
		}
	}
	return fmt.Sprintf("%s%04d", prefix, highest+1)
}
