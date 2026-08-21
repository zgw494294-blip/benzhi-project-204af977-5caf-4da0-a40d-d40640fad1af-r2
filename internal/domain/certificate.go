package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

type CertificateSummary struct {
	Session      CalibrationSession  `json:"session"`
	Samples      []ReferenceSample   `json:"samples"`
	Measurements []MeasurementRecord `json:"measurements"`
	Reviews      []QualityReview     `json:"reviews"`
}

const (
	CertificateVerificationVerified = "verified"
	CertificateVerificationInvalid  = "invalid"
)

type CertificateVerification struct {
	Status          string `json:"status"`
	Verifiable      bool   `json:"verifiable"`
	SummaryVerified bool   `json:"summaryVerified"`
	AuditVerified   bool   `json:"auditVerified"`
	FailureReason   string `json:"failureReason,omitempty"`
}

type CertificateView struct {
	CalibrationCertificate
	Verification CertificateVerification `json:"verification"`
}

type SessionCertificateLookup struct {
	CalibrationSession
	Certificate   CertificateView `json:"certificate"`
	AuditVerified bool            `json:"auditVerified"`
}

func CertificateDigest(session CalibrationSession, samples []ReferenceSample, measurements []MeasurementRecord, reviews []QualityReview) (string, error) {
	payload := CertificateSummary{Session: session, Samples: samples, Measurements: measurements, Reviews: reviews}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}

func VerifyCertificateDigest(certificate CalibrationCertificate, session CalibrationSession, samples []ReferenceSample, measurements []MeasurementRecord, reviews []QualityReview) error {
	digest, err := CertificateDigest(session, samples, measurements, reviews)
	if err != nil {
		return fmt.Errorf("重新计算证书摘要失败: %w", err)
	}
	if digest != certificate.SummaryHash {
		return fmt.Errorf("证书摘要哈希不匹配")
	}
	return nil
}
