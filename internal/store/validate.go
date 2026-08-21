package store

import (
	"fmt"

	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/audit"
	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/domain"
)

func validateLedger(ledger Ledger) error {
	return validateLedgerWithAudit(ledger, true)
}

func validateLedgerForCommit(ledger Ledger) error {
	return validateLedgerWithAudit(ledger, false)
}

func validateLedgerWithAudit(ledger Ledger, verifyAudit bool) error {
	if ledger.SchemaVersion != CurrentSchemaVersion {
		return fmt.Errorf("%w: %d", ErrSchemaVersion, ledger.SchemaVersion)
	}
	if ledger.Revision < 0 {
		return fmt.Errorf("%w: revision 不能为负数", ErrCorruptLedger)
	}
	for id, session := range ledger.Sessions {
		if id != session.ID {
			return fmt.Errorf("%w: 会话键 %s 不匹配", ErrCorruptLedger, id)
		}
		if err := session.Validate(); err != nil {
			return fmt.Errorf("%w: %v", ErrCorruptLedger, err)
		}
		if err := validateSessionChildren(ledger, session); err != nil {
			return err
		}
		if err := validateStatusConsistency(ledger, session); err != nil {
			return err
		}
	}
	for id, certificate := range ledger.Certificates {
		if _, exists := ledger.Sessions[id]; !exists {
			return fmt.Errorf("%w: 证书没有对应会话", ErrCorruptLedger)
		}
		if id != certificate.SessionID {
			return fmt.Errorf("%w: 证书键 %s 不匹配", ErrCorruptLedger, id)
		}
		if err := certificate.Validate(); err != nil {
			return fmt.Errorf("%w: %v", ErrCorruptLedger, err)
		}
	}
	for sessionID, samples := range ledger.Samples {
		if _, exists := ledger.Sessions[sessionID]; !exists {
			return fmt.Errorf("%w: 标准样本存在孤立会话键", ErrCorruptLedger)
		}
		seenIDs := make(map[string]struct{}, len(samples))
		seenNumbers := make(map[string]struct{}, len(samples))
		for _, sample := range samples {
			if _, exists := seenIDs[sample.ID]; exists {
				return fmt.Errorf("%w: 标准样本 ID 重复", ErrCorruptLedger)
			}
			if _, exists := seenNumbers[sample.SampleNumber]; exists {
				return fmt.Errorf("%w: 样本编号重复", ErrCorruptLedger)
			}
			seenIDs[sample.ID] = struct{}{}
			seenNumbers[sample.SampleNumber] = struct{}{}
		}
	}
	for sessionID, measurements := range ledger.Measurements {
		if _, exists := ledger.Sessions[sessionID]; !exists {
			return fmt.Errorf("%w: 测量记录存在孤立会话键", ErrCorruptLedger)
		}
		seenIDs := make(map[string]struct{}, len(measurements))
		seenKeys := make(map[string]struct{}, len(measurements))
		seenSequences := make(map[int]struct{}, len(measurements))
		for _, measurement := range measurements {
			if _, exists := seenIDs[measurement.ID]; exists {
				return fmt.Errorf("%w: 测量记录 ID 重复", ErrCorruptLedger)
			}
			if measurement.IdempotencyKey != "" {
				if _, exists := seenKeys[measurement.IdempotencyKey]; exists {
					return fmt.Errorf("%w: 测量幂等键重复", ErrCorruptLedger)
				}
				seenKeys[measurement.IdempotencyKey] = struct{}{}
			}
			if _, exists := domain.FindSample(ledger.Samples[sessionID], measurement.SampleID); !exists {
				return fmt.Errorf("%w: 测量记录引用不存在的标准样本", ErrCorruptLedger)
			}
			if _, exists := seenSequences[measurement.MeasurementSequence]; exists {
				return fmt.Errorf("%w: 测量序号重复", ErrCorruptLedger)
			}
			seenSequences[measurement.MeasurementSequence] = struct{}{}
			seenIDs[measurement.ID] = struct{}{}
		}
	}
	for sessionID, reviews := range ledger.Reviews {
		if _, exists := ledger.Sessions[sessionID]; !exists {
			return fmt.Errorf("%w: 复核记录存在孤立会话键", ErrCorruptLedger)
		}
		seenIDs := make(map[string]struct{}, len(reviews))
		for _, review := range reviews {
			if _, exists := seenIDs[review.ID]; exists {
				return fmt.Errorf("%w: 复核记录 ID 重复", ErrCorruptLedger)
			}
			seenIDs[review.ID] = struct{}{}
		}
	}
	for sessionID, events := range ledger.Audits {
		if _, exists := ledger.Sessions[sessionID]; !exists {
			return fmt.Errorf("%w: 审计事件存在孤立会话键", ErrCorruptLedger)
		}
		for _, event := range events {
			if event.SessionID != sessionID || event.ID == "" || event.Sequence < 1 {
				return fmt.Errorf("%w: 审计事件字段不完整", ErrCorruptLedger)
			}
		}
		if verifyAudit {
			if err := audit.Verify(events); err != nil {
				return fmt.Errorf("%w: 审计哈希链校验失败: %v", ErrCorruptLedger, err)
			}
		}
	}
	return nil
}

func validateStatusConsistency(ledger Ledger, session domain.CalibrationSession) error {
	samples := ledger.Samples[session.ID]
	measurements := ledger.Measurements[session.ID]
	reviews := ledger.Reviews[session.ID]
	switch session.Status {
	case domain.StatusDraft:
		if len(samples) != 0 || len(measurements) != 0 || len(reviews) != 0 || hasCertificate(ledger, session.ID) {
			return fmt.Errorf("%w: 草稿会话包含已开始的业务记录", ErrCorruptLedger)
		}
	case domain.StatusMeasuring:
		if len(samples) == 0 || !hasWritableMeasurementSet(samples, measurements) || hasCertificate(ledger, session.ID) {
			return fmt.Errorf("%w: 测量中会话的样本或测量状态不一致", ErrCorruptLedger)
		}
	case domain.StatusPendingReview:
		if !domain.AllSamplesMeasured(samples, measurements) || hasCertificate(ledger, session.ID) {
			return fmt.Errorf("%w: 待复核会话存在未测样本", ErrCorruptLedger)
		}
	case domain.StatusRework:
		if !domain.AllSamplesMeasured(samples, measurements) || hasCertificate(ledger, session.ID) {
			return fmt.Errorf("%w: 返工会话存在未完成的初次测量", ErrCorruptLedger)
		}
	case domain.StatusReadyToSeal:
		if !domain.AllSamplesQualified(samples, measurements) || !hasReviewConclusion(reviews, "passed") || hasCertificate(ledger, session.ID) {
			return fmt.Errorf("%w: 可封存会话缺少合格测量或通过复核", ErrCorruptLedger)
		}
	case domain.StatusSealed:
		certificate, ok := ledger.Certificates[session.ID]
		if !ok || certificate.SessionID != session.ID || !domain.AllSamplesQualified(samples, measurements) || !hasReviewConclusion(reviews, "passed") {
			return fmt.Errorf("%w: 已封存会话的证书、测量或复核状态不一致", ErrCorruptLedger)
		}
	default:
		return fmt.Errorf("%w: 会话状态未知", ErrCorruptLedger)
	}
	return nil
}

func hasCertificate(ledger Ledger, sessionID string) bool {
	_, ok := ledger.Certificates[sessionID]
	return ok
}

func hasWritableMeasurementSet(samples []domain.ReferenceSample, measurements []domain.MeasurementRecord) bool {
	return len(samples) > 0 && !domain.AllSamplesQualified(samples, measurements)
}

func hasReviewConclusion(reviews []domain.QualityReview, conclusion string) bool {
	for _, review := range reviews {
		if review.Conclusion == conclusion {
			return true
		}
	}
	return false
}

func validateSessionChildren(ledger Ledger, session domain.CalibrationSession) error {
	for _, sample := range ledger.Samples[session.ID] {
		if sample.SessionID != session.ID {
			return fmt.Errorf("%w: 标准样本不属于会话", ErrCorruptLedger)
		}
		if err := sample.Validate(); err != nil {
			return fmt.Errorf("%w: %v", ErrCorruptLedger, err)
		}
	}
	for _, measurement := range ledger.Measurements[session.ID] {
		if measurement.SessionID != session.ID {
			return fmt.Errorf("%w: 测量记录不属于会话", ErrCorruptLedger)
		}
		if err := measurement.Validate(); err != nil {
			return fmt.Errorf("%w: %v", ErrCorruptLedger, err)
		}
	}
	for _, review := range ledger.Reviews[session.ID] {
		if review.SessionID != session.ID {
			return fmt.Errorf("%w: 复核记录不属于会话", ErrCorruptLedger)
		}
		if err := review.Validate(); err != nil {
			return fmt.Errorf("%w: %v", ErrCorruptLedger, err)
		}
	}
	return nil
}
