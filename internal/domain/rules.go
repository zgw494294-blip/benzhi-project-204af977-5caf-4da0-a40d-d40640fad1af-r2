package domain

import (
	"math"
	"sort"
)

func (s CalibrationSession) Validate() error {
	if s.ID == "" || s.DeviceID == "" || s.DeviceName == "" || s.ObservingBand == "" || s.Owner == "" {
		return Invalid("invalid_session", "会话必须包含设备编号、设备名称、观测波段和负责人")
	}
	if s.Status == "" || s.Version < 1 || s.CreatedAt.IsZero() || s.UpdatedAt.IsZero() {
		return Invalid("invalid_session", "会话状态、版本和时间字段不完整")
	}
	return nil
}

func (s ReferenceSample) Validate() error {
	if s.ID == "" || s.SessionID == "" || s.SampleNumber == "" || s.Unit == "" || s.RegisteredBy == "" {
		return Invalid("invalid_sample", "标准样本必须包含编号、单位和登记人")
	}
	if math.IsNaN(s.ReferenceValue) || math.IsInf(s.ReferenceValue, 0) || s.ReferenceValue < 0 {
		return Invalid("invalid_sample", "参考值必须是非负有限数字")
	}
	if math.IsNaN(s.AllowedDelta) || math.IsInf(s.AllowedDelta, 0) || s.AllowedDelta <= 0 {
		return Invalid("invalid_sample", "允许偏差必须是正数")
	}
	return nil
}

func (m MeasurementRecord) Validate() error {
	if m.ID == "" || m.SessionID == "" || m.SampleID == "" || m.Operator == "" || m.MeasurementSequence < 1 {
		return Invalid("invalid_measurement", "测量记录字段不完整或序号无效")
	}
	if math.IsNaN(m.MeasuredValue) || math.IsInf(m.MeasuredValue, 0) || m.MeasuredValue < 0 {
		return Invalid("invalid_measurement", "测量值必须是非负有限数字")
	}
	return nil
}

func (r QualityReview) Validate() error {
	if r.ID == "" || r.SessionID == "" || r.Reviewer == "" || r.ReviewedAt.IsZero() {
		return Invalid("invalid_review", "复核记录字段不完整")
	}
	if r.Conclusion != "passed" && r.Conclusion != "rework" {
		return Invalid("invalid_review", "复核结论必须是 passed 或 rework")
	}
	if r.Conclusion == "rework" && r.ReworkReason == "" {
		return Invalid("invalid_review", "退回返工必须填写原因")
	}
	return nil
}

func (c CalibrationCertificate) Validate() error {
	if c.ID == "" || c.SessionID == "" || c.CertificateNo == "" || c.SummaryHash == "" || c.SealedBy == "" || c.SealedAt.IsZero() {
		return Invalid("invalid_certificate", "校准证书字段不完整")
	}
	return nil
}

func FindSample(samples []ReferenceSample, id string) (ReferenceSample, bool) {
	for _, sample := range samples {
		if sample.ID == id {
			return sample, true
		}
	}
	return ReferenceSample{}, false
}

func LatestMeasurement(measurements []MeasurementRecord, sampleID string) (MeasurementRecord, bool) {
	var latest MeasurementRecord
	found := false
	for _, measurement := range measurements {
		if measurement.SampleID == sampleID && (!found || measurement.MeasurementSequence > latest.MeasurementSequence) {
			latest = measurement
			found = true
		}
	}
	return latest, found
}

func NextSampleID(samples []ReferenceSample, measurements []MeasurementRecord) string {
	ordered := append([]ReferenceSample(nil), samples...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].RegisteredAt.Before(ordered[j].RegisteredAt) })
	for _, sample := range ordered {
		if _, ok := LatestMeasurement(measurements, sample.ID); !ok {
			return sample.ID
		}
	}
	for _, sample := range ordered {
		latest, _ := LatestMeasurement(measurements, sample.ID)
		if !latest.WithinTolerance {
			return sample.ID
		}
	}
	return ""
}

func AllSamplesMeasured(samples []ReferenceSample, measurements []MeasurementRecord) bool {
	if len(samples) == 0 {
		return false
	}
	for _, sample := range samples {
		if _, ok := LatestMeasurement(measurements, sample.ID); !ok {
			return false
		}
	}
	return true
}

func AllSamplesQualified(samples []ReferenceSample, measurements []MeasurementRecord) bool {
	if !AllSamplesMeasured(samples, measurements) {
		return false
	}
	for _, sample := range samples {
		latest, _ := LatestMeasurement(measurements, sample.ID)
		if !latest.WithinTolerance {
			return false
		}
	}
	return true
}

func DeviationFor(sample ReferenceSample, measured float64) float64 {
	return measured - sample.ReferenceValue
}
