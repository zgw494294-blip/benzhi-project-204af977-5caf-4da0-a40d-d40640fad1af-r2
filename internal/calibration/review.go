package calibration

import (
	"context"
	"strings"

	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/audit"
	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/domain"
	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/store"
)

func (s *Service) SubmitReview(sessionID string, request ReviewInput) (domain.CalibrationSession, domain.QualityReview, error) {
	return s.SubmitReviewContext(context.Background(), sessionID, request)
}

func (s *Service) SubmitReviewContext(ctx context.Context, sessionID string, request ReviewInput) (domain.CalibrationSession, domain.QualityReview, error) {
	request.Reviewer = strings.TrimSpace(request.Reviewer)
	request.Conclusion = normalizeConclusion(request.Conclusion)
	request.ReworkReason = strings.TrimSpace(request.ReworkReason)
	if request.ExpectedVersion < 1 || request.Reviewer == "" || request.Conclusion == "" {
		return domain.CalibrationSession{}, domain.QualityReview{}, invalid("invalid_request", "expectedVersion、reviewer 和有效 conclusion 均为必填")
	}
	now := s.now().UTC()
	var sessionResult domain.CalibrationSession
	var review domain.QualityReview
	err := s.store.UpdateContext(ctx, func(ledger *store.Ledger) error {
		session, ok := ledger.Sessions[sessionID]
		if !ok {
			return notFound("校准会话")
		}
		if session.Version != request.ExpectedVersion {
			return invalid("version_conflict", fmtVersion(session.Version))
		}
		if !domain.CanReview(session.Status) {
			return invalid("invalid_state", "只有完成全部样本测量后才能复核")
		}
		samples := ledger.Samples[sessionID]
		measurements := ledger.Measurements[sessionID]
		qualified := domain.AllSamplesQualified(samples, measurements)
		if request.Conclusion == "passed" && !qualified {
			return invalid("quality_failed", "存在超出允许偏差的样本，只能退回返工")
		}
		if request.Conclusion == "rework" && request.ReworkReason == "" {
			return invalid("invalid_request", "退回返工必须填写原因")
		}
		review = domain.QualityReview{ID: s.newID("REV"), SessionID: sessionID, Reviewer: request.Reviewer, Conclusion: request.Conclusion, DeviationSummary: deviationSummary(samples, measurements), ReworkReason: request.ReworkReason, ReviewedAt: now}
		if err := review.Validate(); err != nil {
			return err
		}
		ledger.Reviews[sessionID] = append(ledger.Reviews[sessionID], review)
		if request.Conclusion == "passed" {
			session.Status = domain.StatusReadyToSeal
		} else {
			session.Status = domain.StatusRework
		}
		session.Version++
		session.UpdatedAt = now
		ledger.Sessions[sessionID] = session
		audit.Append(&ledger.Audits, audit.NewEvent(s.newID("AUD"), sessionID, "quality.reviewed", review.Reviewer, map[string]string{"conclusion": review.Conclusion, "qualified": fmtBool(qualified)}, now))
		sessionResult = session
		return nil
	})
	if err != nil {
		return domain.CalibrationSession{}, domain.QualityReview{}, err
	}
	return sessionResult, review, nil
}

func deviationSummary(samples []domain.ReferenceSample, measurements []domain.MeasurementRecord) string {
	failed := make([]string, 0)
	for _, measurement := range measurements {
		if measurement.WithinTolerance {
			continue
		}
		sample, ok := domain.FindSample(samples, measurement.SampleID)
		if ok {
			failed = append(failed, sample.SampleNumber+" 偏差 "+formatFloat(measurement.Deviation)+" "+sample.Unit)
		}
	}
	if len(failed) == 0 {
		return "全部样本均在允许偏差内"
	}
	return strings.Join(failed, "；")
}
