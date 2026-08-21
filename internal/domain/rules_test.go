package domain

import (
	"testing"
	"time"
)

func TestNextSampleIDFollowsOrderAndReturnsFailedSample(t *testing.T) {
	now := time.Date(2026, 8, 20, 8, 0, 0, 0, time.UTC)
	samples := []ReferenceSample{
		{ID: "s1", SessionID: "session", SampleNumber: "REF-01", ReferenceValue: 10, Unit: "ADU", AllowedDelta: 1, RegisteredBy: "工程师", RegisteredAt: now},
		{ID: "s2", SessionID: "session", SampleNumber: "REF-02", ReferenceValue: 20, Unit: "ADU", AllowedDelta: 1, RegisteredBy: "工程师", RegisteredAt: now.Add(time.Minute)},
	}
	measurements := []MeasurementRecord{{ID: "m1", SessionID: "session", SampleID: "s1", MeasurementSequence: 1, MeasuredValue: 10, Operator: "值班员", WithinTolerance: true}}
	if got := NextSampleID(samples, measurements); got != "s2" {
		t.Fatalf("expected s2, got %s", got)
	}
	measurements = append(measurements, MeasurementRecord{ID: "m2", SessionID: "session", SampleID: "s2", MeasurementSequence: 2, MeasuredValue: 23, Operator: "值班员", WithinTolerance: false})
	if got := NextSampleID(samples, measurements); got != "s2" {
		t.Fatalf("expected failed s2 to be retried, got %s", got)
	}
}

func TestCertificateDigestChangesWhenMeasurementChanges(t *testing.T) {
	now := time.Date(2026, 8, 20, 8, 0, 0, 0, time.UTC)
	session := CalibrationSession{ID: "session", DeviceID: "AST-1", DeviceName: "光谱仪", ObservingBand: "可见光", Owner: "工程师", Status: StatusReadyToSeal, Version: 3, CreatedAt: now, UpdatedAt: now}
	sample := ReferenceSample{ID: "sample", SessionID: "session", SampleNumber: "REF-01", ReferenceValue: 10, Unit: "ADU", AllowedDelta: 1, RegisteredBy: "工程师", RegisteredAt: now}
	measurement := MeasurementRecord{ID: "measurement", SessionID: "session", SampleID: "sample", MeasuredValue: 10, MeasurementSequence: 1, Operator: "值班员", WithinTolerance: true}
	first, err := CertificateDigest(session, []ReferenceSample{sample}, []MeasurementRecord{measurement}, nil)
	if err != nil {
		t.Fatal(err)
	}
	measurement.MeasuredValue = 10.5
	second, err := CertificateDigest(session, []ReferenceSample{sample}, []MeasurementRecord{measurement}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("digest did not change with measurement")
	}
}
